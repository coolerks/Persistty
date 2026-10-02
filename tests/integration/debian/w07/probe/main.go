// W07 acceptance client. It never installs policy or invokes sudo directly.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"golang.org/x/term"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"persistty/internal/auth"
	"persistty/internal/elevation"
	"persistty/internal/files"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type client struct {
	origin string
	csrf   string
	http   *http.Client
}

func (c *client) call(ctx context.Context, method, path string, input, output any) (int, error) {
	var data []byte
	var err error
	if input != nil {
		data, err = json.Marshal(input)
		if err != nil {
			return 0, err
		}
		defer clear(data)
	}
	request, err := http.NewRequestWithContext(ctx, method, c.origin+path, bytes.NewReader(data))
	if err != nil {
		return 0, errors.New("请求构造失败")
	}
	request.Header.Set("Origin", c.origin)
	request.Header.Set("Content-Type", "application/json")
	if c.csrf != "" {
		request.Header.Set("X-CSRF-Token", c.csrf)
	}
	response, err := c.http.Do(request)
	if err != nil {
		return 0, errors.New("请求失败，不重放写入；请人工查询本次结果")
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 6*elevation.MaxContent+8193))
	defer clear(body)
	if err != nil {
		return response.StatusCode, errors.New("读取响应失败")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return response.StatusCode, nil
	}
	if output != nil {
		var envelope struct {
			Data      json.RawMessage `json:"data"`
			RequestID string          `json:"request_id"`
		}
		if json.Unmarshal(body, &envelope) != nil || json.Unmarshal(envelope.Data, output) != nil {
			return response.StatusCode, errors.New("响应解码失败")
		}
	}
	return response.StatusCode, nil
}
func password(label string) ([]byte, error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return nil, errors.New("必须从本机 TTY 隐藏输入，不接受 argv/env/文件密码")
	}
	fmt.Fprint(os.Stderr, label)
	value, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return nil, errors.New("隐藏输入失败")
	}
	return value, nil
}
func run() error {
	origin := flag.String("origin", "", "已批准隔离实例的 origin")
	project := flag.String("project", "", "项目 ID")
	folder := flag.String("folder", "", "folder ID")
	relative := flag.String("path", "", "隔离文本相对路径")
	version := flag.Int64("project-version", 0, "当前配置版本")
	expect := flag.String("expect", "applied", "applied/rejected/cancelled")
	ack := flag.Bool("ack-write-isolated-target", false, "明确允许修改该隔离目标")
	flag.Parse()
	parsed, err := url.Parse(*origin)
	validID := regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)
	if err != nil || parsed.User != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" || !validID.MatchString(*project) || !validID.MatchString(*folder) || *version < 1 || !files.ValidRelative(*relative, false) || !*ack || flag.NArg() != 0 {
		return errors.New("需要精确 origin/project/folder/path/version 和隔离目标写入确认；HTTP 仅限已批准 VPN")
	}
	if *expect != "applied" && *expect != "rejected" && *expect != "cancelled" {
		return errors.New("无效 expect")
	}
	app, err := password("应用登录密码（隐藏）: ")
	if err != nil {
		return err
	}
	defer clear(app)
	jar, _ := cookiejar.New(nil)
	c := client{origin: *origin, http: &http.Client{Jar: jar, Timeout: 40 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("拒绝凭据重定向") }, Transport: &http.Transport{Proxy: nil}}}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	var session auth.Session
	status, err := c.call(ctx, "POST", "/api/v1/auth/login", map[string]string{"password": string(app)}, &session)
	clear(app)
	if err != nil {
		return err
	}
	if status != 200 {
		return errors.New("应用登录未成功")
	}
	c.csrf = session.CSRFToken
	defer c.call(context.Background(), "POST", "/api/v1/auth/logout", nil, nil)
	var system []byte
	if *expect != "cancelled" {
		system, err = password("当前系统账户密码（隐藏；错误口令实验请主动输入错误值）: ")
		if err != nil {
			return err
		}
		defer clear(system)
	}
	base := "/api/v1/projects/" + *project + "/folders/" + *folder
	query := url.Values{"project_version": {strconv.FormatInt(*version, 10)}, "path": {*relative}}.Encode()
	var snapshot files.Content
	status, err = c.call(ctx, "GET", base+"/content?"+query, nil, &snapshot)
	if err != nil {
		return err
	}
	if status != 200 {
		return errors.New("普通账户必须能读取该已有文本")
	}
	content := snapshot.Content + "\nW07 acceptance isolated marker\n"
	var prepared elevation.Prepared
	input := map[string]any{"project_version": *version, "path": *relative, "expected_version": snapshot.Version, "content": content}
	status, err = c.call(ctx, "POST", base+"/elevation-requests", input, &prepared)
	delete(input, "content")
	if err != nil {
		return err
	}
	if status != 200 {
		return errors.New("prepare 未成功，请核对允许列表和版本")
	}
	path := "/api/v1/elevation-requests/" + prepared.ID
	fmt.Printf("request_id=%s target_id=%s\n", prepared.ID, prepared.TargetID)
	var result elevation.Result
	if *expect == "cancelled" {
		status, err = c.call(ctx, "DELETE", path, nil, &result)
	} else {
		status, err = c.call(ctx, "POST", path+"/execute", map[string]string{"password": string(system), "content": content}, &result)
		clear(system)
	}
	if err != nil {
		return err
	}
	if status != 200 || !elevation.ResultValid(result, prepared.ID) || result.State != *expect {
		return errors.New("实际保存状态与期望不符；不得重放本次请求")
	}
	fmt.Printf("state=%s\n", result.State)
	status, err = c.call(ctx, "POST", path+"/execute", map[string]string{"password": "synthetic-replay-denied", "content": content}, nil)
	if err != nil {
		return err
	}
	if status != 409 {
		return errors.New("重复 execute 未拒绝")
	}
	var actual files.Content
	status, err = c.call(ctx, "GET", base+"/content?"+query, nil, &actual)
	if err != nil {
		return err
	}
	if status != 200 {
		return errors.New("保存后普通复读失败")
	}
	want := snapshot.Content
	if *expect == "applied" {
		want = content
	}
	if actual.Content != want {
		return errors.New("目标正文不符合预期")
	}
	fmt.Println("replay_rejected=true content_verified=true")
	return nil
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "W07:", strings.TrimSpace(err.Error()))
		os.Exit(1)
	}
}
