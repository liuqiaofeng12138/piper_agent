# pkg/chrome

浏览器自动化（非 Route 层）。

## 子目录

- [action](action/README.md) — Click、Scroll、Screenshot、Exec 等
- [action/login](action/login/README.md) — Login、LoginManuallyCheck
- [filter](filter/README.md) — ProxyRequest/ResponseFilter
- [phase](phase/README.md) — Init、Idle、HangUp、Phrase

## 其他

- `ChromeDistributor` 逻辑主要在 `pkg/agent/chrome` 与 `pkg/distributor` 衔接
- `WebDriverUtil`, `BMProxyBuilder`, `DocumentSettleCondition`

