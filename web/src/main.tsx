import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { BrowserRouter } from "react-router";
import { App } from "./app/App";
import { applyTheme, readTheme } from "./features/settings/theme";
import "./app/styles.css";

applyTheme(readTheme());
const root = document.getElementById("root");
if (!root) throw new Error("缺少应用根节点");
createRoot(root).render(<StrictMode><BrowserRouter><App /></BrowserRouter></StrictMode>);
