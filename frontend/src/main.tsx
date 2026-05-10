import { render } from "preact";
import { App } from "./App";
import "./tokens.css";
import "./styles.css";

const root = document.getElementById("app");
if (!root) {
  throw new Error("#app mount point missing from index.html");
}
render(<App />, root);
