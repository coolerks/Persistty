# W05 合成图片

六种格式均为自有合成 fixture：3×2 的红/绿/蓝/白/黑/黄像素，SVG 为3×2红色矩形。PNG/JPEG/GIF/WebP/AVIF 用 Pillow 12.3.0 编码，SVG为手写静态白名单内容，不来源于用户图片或第三方素材。

用于 `web/tests/e2e/w05-platform-matrix.spec.ts` 验证内容识别、原字节 HTTP 响应、浏览器真实解码及尺寸。测试把文件命名为 `.txt`，确认后缀不决定预览类型；无需在运行测试时安装 Pillow。
