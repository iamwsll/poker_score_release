# 后端接口测试使用说明

## 推荐流程：Go 集成测试
1. 确保后端可以启动所需的系统依赖（无需手动运行服务，`go test` 会使用内存数据库 + `httptest`）。
2. 在项目根目录执行：
   ```bash
   cd backend
   go test ./...
   ```
3. 关注 `controllers.TestFullIntegrationFlow` 输出，它会覆盖注册、房间、操作、结算、战绩、后台等完整流程。
4. 所有断言基于 Go 原生测试框架与 `github.com/stretchr/testify/require`，失败会直接在终端给出详细断言信息。

### 覆盖范围
- 用户注册 / 登录 / 个人信息 / 会话注销
- 房间创建、加入、返回、踢人、积分操作（下注、牛牛下注、收回、强制转移）
- 结算发起与确认、积分守恒验证
- 今晚战绩及时间区间查询
- 后台权限校验、用户信息更新、房间与结算/进出历史查询
- 结算服务单测补充（`backend/services/settlement_service_test.go`）：校验 `calculateRmbAmount` 的倍率换算逻辑以及 `generateSettlementPlan` 的出入账排序/描述，确保服务层回归无需跑完整集成流程。

## 可选：Legacy Python 回归脚本
旧版 `test_api.py` 仍保留用于真实 HTTP 环境的端到端冒烟测试，需手动启动后端并连接本地 `http://localhost:8080`。

```bash
cd test
pip install -r requirements_test.txt
python3 test_api.py
```

> 提示：Python 脚本会直接操作 `backend/database.db` 提升管理员角色，仅适用于本地测试库。

## 常见问题
- **`go test` 提示端口被占用**：确认没有其它进程监听 8080（集成测试默认在内存模式运行，不会绑定端口）。
- **需要模拟 WebSocket**：目前仅支持手工或专用工具（如 `wscat`），Go 测试聚焦 REST 场景。
- **仍需旧版报告**：请在 Git 历史中查阅 2025-11-07 之前的 `docs/test_report.md`。
