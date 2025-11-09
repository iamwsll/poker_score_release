#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
扑克记分系统后端接口测试脚本
Author: AI Assistant
Date: 2025-11-06
"""

import requests
import json
import time
from datetime import datetime
from typing import Dict, Any, Optional
import sys

# 配置
BASE_URL = "http://localhost:8080/api"
TEST_REPORT_FILE = "../docs/test_report.md"

# 全局变量存储测试数据
test_data = {
    "users": [],
    "rooms": [],
    "session_cookies": {},
    "admin_cookie": None
}

# 测试结果统计
test_stats = {
    "total": 0,
    "passed": 0,
    "failed": 0,
    "errors": []
}

# 测试报告内容
report_lines = []


def log(message: str, level: str = "INFO"):
    """打印日志"""
    timestamp = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
    print(f"[{timestamp}] [{level}] {message}")


def add_report_line(line: str):
    """添加到测试报告"""
    report_lines.append(line)


def test_api(name: str, method: str, url: str, 
             data: Optional[Dict] = None, 
             cookies: Optional[Dict] = None,
             expected_code: int = 0,
             expected_status: int = 200) -> Optional[Dict]:
    """
    测试API接口的通用函数
    
    Args:
        name: 测试名称
        method: HTTP方法 (GET, POST, PUT, DELETE)
        url: 接口URL
        data: 请求数据
        cookies: Cookie数据
        expected_code: 期望的业务code (默认0表示成功)
        expected_status: 期望的HTTP状态码
    
    Returns:
        响应数据,如果失败返回None
    """
    test_stats["total"] += 1
    
    try:
        log(f"开始测试: {name}")
        add_report_line(f"\n#### {name}\n")
        add_report_line(f"**请求**: `{method} {url}`\n")
        
        if data:
            add_report_line(f"**请求数据**:\n```json\n{json.dumps(data, ensure_ascii=False, indent=2)}\n```\n")
        
        # 发送请求
        full_url = f"{BASE_URL}{url}" if not url.startswith("http") else url
        
        if method == "GET":
            response = requests.get(full_url, cookies=cookies, timeout=10)
        elif method == "POST":
            response = requests.post(full_url, json=data, cookies=cookies, timeout=10)
        elif method == "PUT":
            response = requests.put(full_url, json=data, cookies=cookies, timeout=10)
        elif method == "DELETE":
            response = requests.delete(full_url, cookies=cookies, timeout=10)
        else:
            raise ValueError(f"不支持的HTTP方法: {method}")
        
        # 检查HTTP状态码
        add_report_line(f"**HTTP状态码**: {response.status_code}\n")
        
        # 解析响应
        try:
            resp_data = response.json()
            add_report_line(f"**响应数据**:\n```json\n{json.dumps(resp_data, ensure_ascii=False, indent=2)}\n```\n")
        except:
            add_report_line(f"**响应数据**: {response.text}\n")
            resp_data = None
        
        # 验证结果
        if response.status_code != expected_status:
            log(f"✗ 测试失败: {name} - HTTP状态码不匹配 (期望: {expected_status}, 实际: {response.status_code})", "ERROR")
            add_report_line(f"**测试结果**: ❌ 失败 - HTTP状态码不匹配 (期望: {expected_status}, 实际: {response.status_code})\n")
            test_stats["failed"] += 1
            test_stats["errors"].append(f"{name}: HTTP状态码不匹配")
            return None
        
        if resp_data and "code" in resp_data:
            if resp_data["code"] != expected_code:
                log(f"✗ 测试失败: {name} - 业务code不匹配 (期望: {expected_code}, 实际: {resp_data['code']}, 消息: {resp_data.get('message', '')})", "ERROR")
                add_report_line(f"**测试结果**: ❌ 失败 - 业务code不匹配 (期望: {expected_code}, 实际: {resp_data['code']})\n")
                add_report_line(f"**错误信息**: {resp_data.get('message', '')}\n")
                test_stats["failed"] += 1
                test_stats["errors"].append(f"{name}: {resp_data.get('message', '')}")
                return None
        
        log(f"✓ 测试通过: {name}", "SUCCESS")
        add_report_line(f"**测试结果**: ✅ 通过\n")
        test_stats["passed"] += 1
        
        # 保存Set-Cookie
        if "Set-Cookie" in response.headers:
            log(f"  收到Cookie: {response.headers['Set-Cookie'][:50]}...")
        
        return resp_data
        
    except Exception as e:
        log(f"✗ 测试异常: {name} - {str(e)}", "ERROR")
        add_report_line(f"**测试结果**: ❌ 异常 - {str(e)}\n")
        test_stats["failed"] += 1
        test_stats["errors"].append(f"{name}: {str(e)}")
        return None


def test_auth_apis():
    """测试用户认证相关接口"""
    log("=" * 60)
    log("开始测试用户认证相关接口")
    add_report_line("\n## 一、用户认证相关接口测试\n")
    
    # 1.1 用户注册
    user1_phone = f"138{int(time.time()) % 100000000:08d}"
    resp = test_api(
        "1.1 用户注册",
        "POST",
        "/auth/register",
        data={
            "phone": user1_phone,
            "nickname": "测试用户1",
            "password": "123456"
        }
    )
    
    if resp and resp.get("data"):
        test_data["users"].append({
            "id": resp["data"]["user"]["id"],
            "phone": user1_phone,
            "nickname": "测试用户1",
            "password": "123456",
            "session_id": resp["data"].get("session_id")
        })
        log(f"  用户ID: {resp['data']['user']['id']}")
    
    # 获取cookie用于后续测试 - 注册用户2
    user2_phone = f"139{int(time.time()) % 100000000:08d}"
    session = requests.Session()
    resp_obj = session.post(f"{BASE_URL}/auth/register", json={
        "phone": user2_phone,
        "nickname": "测试用户2",
        "password": "123456"
    })
    if resp_obj.status_code == 200:
        resp_data = resp_obj.json()
        if resp_data.get("data"):
            test_data["users"].append({
                "id": resp_data["data"]["user"]["id"],
                "phone": user2_phone,
                "nickname": "测试用户2",
                "password": "123456",
                "session_id": resp_data["data"].get("session_id")
            })
            test_data["session_cookies"]["user2"] = session.cookies.get_dict()
            log(f"  创建测试用户2, ID: {resp_data['data']['user']['id']}, Cookie: {session.cookies.get_dict()}")
    
    # 1.2 用户登录
    resp = test_api(
        "1.2 用户登录",
        "POST",
        "/auth/login",
        data={
            "phone": user1_phone,
            "password": "123456"
        }
    )
    
    if resp and resp.get("data"):
        log(f"  登录成功,用户ID: {resp['data']['user']['id']}")
        # 获取登录后的cookie
        session_login = requests.Session()
        session_login.post(f"{BASE_URL}/auth/login", json={
            "phone": user1_phone,
            "password": "123456"
        })
        test_data["session_cookies"]["user1"] = session_login.cookies.get_dict()
        log(f"  保存用户1登录Cookie: {test_data['session_cookies']['user1']}")
    
    # 1.3 获取当前用户信息
    test_api(
        "1.3 获取当前用户信息",
        "GET",
        "/auth/me",
        cookies=test_data["session_cookies"].get("user2")
    )
    
    # 1.4 修改昵称
    test_api(
        "1.4 修改昵称",
        "PUT",
        "/auth/nickname",
        data={"nickname": "新昵称测试"},
        cookies=test_data["session_cookies"].get("user2")
    )
    
    # 1.5 修改密码
    test_api(
        "1.5 修改密码",
        "PUT",
        "/auth/password",
        data={
            "old_password": "123456",
            "new_password": "654321"
        },
        cookies=test_data["session_cookies"].get("user2")
    )
    
    # 1.6 登出
    test_api(
        "1.6 用户登出",
        "POST",
        "/auth/logout",
        cookies=test_data["session_cookies"].get("user2")
    )
    
    # 1.7 测试错误场景:未登录访问需要认证的接口
    test_api(
        "1.7 未登录访问认证接口(应失败)",
        "GET",
        "/auth/me",
        expected_code=401,
        expected_status=401
    )


def test_room_apis():
    """测试房间管理相关接口"""
    log("=" * 60)
    log("开始测试房间管理相关接口")
    add_report_line("\n## 二、房间管理相关接口测试\n")
    
    # 先登录获取cookie
    if not test_data["session_cookies"].get("user1"):
        log("需要先完成用户认证测试", "WARNING")
        return
    
    # 2.1 创建房间
    resp = test_api(
        "2.1 创建房间(德扑)",
        "POST",
        "/rooms",
        data={
            "room_type": "texas",
            "chip_rate": "20:1"
        },
        cookies=test_data["session_cookies"].get("user1")
    )
    
    if resp and resp.get("data"):
        test_data["rooms"].append({
            "id": resp["data"]["room_id"],
            "code": resp["data"]["room_code"],
            "type": "texas"
        })
        log(f"  房间ID: {resp['data']['room_id']}, 房间号: {resp['data']['room_code']}")
    
    # 2.2 加入房间
    if test_data["rooms"]:
        room_code = test_data["rooms"][0]["code"]
        # 创建另一个用户来加入房间
        session3 = requests.Session()
        resp_obj = session3.post(f"{BASE_URL}/auth/register", json={
            "phone": f"137{int(time.time()) % 100000000:08d}",
            "nickname": "测试用户3",
            "password": "123456"
        })
        if resp_obj.status_code == 200:
            test_data["session_cookies"]["user3"] = session3.cookies.get_dict()
            
            test_api(
                "2.2 加入房间",
                "POST",
                "/rooms/join",
                data={"room_code": room_code},
                cookies=test_data["session_cookies"].get("user3")
            )
    
    # 2.3 获取房间详情
    if test_data["rooms"]:
        room_id = test_data["rooms"][0]["id"]
        test_api(
            "2.3 获取房间详情",
            "GET",
            f"/rooms/{room_id}",
            cookies=test_data["session_cookies"].get("user1")
        )
    
    # 2.4 返回上次房间
    test_api(
        "2.4 返回上次房间",
        "GET",
        "/rooms/last",
        cookies=test_data["session_cookies"].get("user1")
    )
    
    # 2.5 踢出用户
    if test_data["rooms"] and len(test_data["users"]) >= 3:
        room_id = test_data["rooms"][0]["id"]
        user3_id = test_data["users"][2]["id"]
        test_api(
            "2.5 踢出用户",
            "POST",
            f"/rooms/{room_id}/kick",
            data={"user_id": user3_id},
            cookies=test_data["session_cookies"].get("user1")
        )


def test_operation_apis():
    """测试房间操作相关接口"""
    log("=" * 60)
    log("开始测试房间操作相关接口")
    add_report_line("\n## 三、房间操作相关接口测试\n")
    
    if not test_data["rooms"]:
        log("需要先创建房间", "WARNING")
        return
    
    room_id = test_data["rooms"][0]["id"]
    
    # 3.1 下注
    test_api(
        "3.1 德扑下注",
        "POST",
        f"/rooms/{room_id}/bet",
        data={"amount": 100},
        cookies=test_data["session_cookies"].get("user1")
    )
    
    # 3.2 再次下注
    test_api(
        "3.2 德扑再次下注",
        "POST",
        f"/rooms/{room_id}/bet",
        data={"amount": 200},
        cookies=test_data["session_cookies"].get("user1")
    )
    
    # 3.3 收回
    test_api(
        "3.3 收回积分",
        "POST",
        f"/rooms/{room_id}/withdraw",
        data={"amount": 150},
        cookies=test_data["session_cookies"].get("user1")
    )
    
    # 3.4 全收(amount=0)
    test_api(
        "3.4 全收积分",
        "POST",
        f"/rooms/{room_id}/withdraw",
        data={"amount": 0},
        cookies=test_data["session_cookies"].get("user1")
    )
    
    # 3.5 获取操作历史
    test_api(
        "3.5 获取操作历史",
        "GET",
        f"/rooms/{room_id}/operations?limit=10&offset=0",
        cookies=test_data["session_cookies"].get("user1")
    )
    
    # 3.6 获取历史金额
    test_api(
        "3.6 获取用户历史操作金额",
        "GET",
        f"/rooms/{room_id}/history-amounts",
        cookies=test_data["session_cookies"].get("user1")
    )


def test_niuniu_operations():
    """测试牛牛操作"""
    log("=" * 60)
    log("开始测试牛牛操作")
    add_report_line("\n### 牛牛房间操作测试\n")
    
    # 创建牛牛房间
    resp = test_api(
        "3.7 创建牛牛房间",
        "POST",
        "/rooms",
        data={
            "room_type": "niuniu",
            "chip_rate": "10:1"
        },
        cookies=test_data["session_cookies"].get("user1")
    )
    
    if resp and resp.get("data"):
        niuniu_room_id = resp["data"]["room_id"]
        
        # 其他用户加入
        if test_data["session_cookies"].get("user3"):
            test_api(
                "3.8 其他用户加入牛牛房间",
                "POST",
                "/rooms/join",
                data={"room_code": resp["data"]["room_code"]},
                cookies=test_data["session_cookies"].get("user3")
            )
        
        # 牛牛下注
        if len(test_data["users"]) >= 2:
            test_api(
                "3.9 牛牛下注",
                "POST",
                f"/rooms/{niuniu_room_id}/niuniu-bet",
                data={
                    "bets": [
                        {
                            "to_user_id": test_data["users"][1]["id"],
                            "amount": 50
                        }
                    ]
                },
                cookies=test_data["session_cookies"].get("user1")
            )


def test_settlement_apis():
    """测试结算相关接口"""
    log("=" * 60)
    log("开始测试结算相关接口")
    add_report_line("\n## 四、结算相关接口测试\n")
    
    if not test_data["rooms"]:
        log("需要先创建房间", "WARNING")
        return
    
    room_id = test_data["rooms"][0]["id"]
    
    # 4.1 发起结算
    resp = test_api(
        "4.1 发起结算",
        "POST",
        f"/rooms/{room_id}/settlement/initiate",
        cookies=test_data["session_cookies"].get("user1")
    )
    
    # 4.2 确认结算
    if resp and resp.get("data", {}).get("can_settle"):
        test_api(
            "4.2 确认结算",
            "POST",
            f"/rooms/{room_id}/settlement/confirm",
            cookies=test_data["session_cookies"].get("user1")
        )


def test_record_apis():
    """测试战绩统计相关接口"""
    log("=" * 60)
    log("开始测试战绩统计相关接口")
    add_report_line("\n## 五、战绩统计相关接口测试\n")
    
    # 5.1 查询今晚战绩
    test_api(
        "5.1 查询今晚战绩",
        "GET",
        "/records/tonight",
        cookies=test_data["session_cookies"].get("user1")
    )
    
    # 5.2 查询指定时间范围战绩
    test_api(
        "5.2 查询指定时间范围战绩",
        "GET",
        "/records/tonight?start_time=2025-11-06T00:00:00Z&end_time=2025-11-07T00:00:00Z",
        cookies=test_data["session_cookies"].get("user1")
    )


def test_admin_apis():
    """测试后台管理相关接口"""
    log("=" * 60)
    log("开始测试后台管理相关接口")
    add_report_line("\n## 六、后台管理相关接口测试\n")
    
    # 6.1 普通用户访问管理接口(应失败)
    test_api(
        "6.1 普通用户访问管理接口(应失败)",
        "GET",
        "/admin/users",
        cookies=test_data["session_cookies"].get("user1"),
        expected_code=403,
        expected_status=403
    )
    
    # 注册管理员用户
    admin_phone = f"188{int(time.time()) % 100000000:08d}"
    session_admin = requests.Session()
    resp_obj = session_admin.post(f"{BASE_URL}/auth/register", json={
        "phone": admin_phone,
        "nickname": "管理员",
        "password": "admin123"
    })
    
    if resp_obj.status_code == 200:
        resp_data = resp_obj.json()
        admin_user_id = resp_data.get("data", {}).get("user", {}).get("id")
        test_data["session_cookies"]["admin"] = session_admin.cookies.get_dict()
        
        # 自动提升为管理员
        log(f"正在将手机号 {admin_phone} (用户ID: {admin_user_id}) 提升为管理员...", "INFO")
        add_report_line(f"\n**注意**: 自动将手机号 {admin_phone} 的用户角色改为 'admin'\n")
        
        try:
            import sqlite3
            conn = sqlite3.connect('../backend/database.db')
            cursor = conn.cursor()
            cursor.execute("UPDATE users SET role = 'admin' WHERE phone = ?", (admin_phone,))
            conn.commit()
            conn.close()
            log(f"✓ 成功提升用户为管理员", "SUCCESS")
            
            # 重新登录以获取新的session
            time.sleep(0.5)
            session_admin = requests.Session()
            session_admin.post(f"{BASE_URL}/auth/login", json={
                "phone": admin_phone,
                "password": "admin123"
            })
            test_data["session_cookies"]["admin"] = session_admin.cookies.get_dict()
            
        except Exception as e:
            log(f"✗ 提升管理员权限失败: {str(e)}", "WARNING")
            add_report_line(f"**警告**: 自动提升管理员权限失败: {str(e)}\n")
        
        # 6.2 获取所有用户列表
        test_api(
            "6.2 获取所有用户列表",
            "GET",
            "/admin/users?page=1&page_size=20",
            cookies=test_data["session_cookies"].get("admin")
        )
        
        # 6.3 获取所有房间列表
        test_api(
            "6.3 获取所有房间列表",
            "GET",
            "/admin/rooms?status=all&page=1&page_size=20",
            cookies=test_data["session_cookies"].get("admin")
        )
        
        # 6.4 获取房间详细信息
        if test_data["rooms"]:
            room_id = test_data["rooms"][0]["id"]
            test_api(
                "6.4 获取房间详细信息(管理员)",
                "GET",
                f"/admin/rooms/{room_id}",
                cookies=test_data["session_cookies"].get("admin")
            )
        
        # 6.5 获取用户历史盈亏
        if test_data["users"]:
            user_id = test_data["users"][0]["id"]
            test_api(
                "6.5 获取用户历史盈亏",
                "GET",
                f"/admin/users/{user_id}/settlements",
                cookies=test_data["session_cookies"].get("admin")
            )
        
        # 6.6 获取用户进出房间历史
        test_api(
            "6.6 获取用户进出房间历史",
            "GET",
            "/admin/room-member-history?page=1&page_size=50",
            cookies=test_data["session_cookies"].get("admin")
        )


def test_health_check():
    """测试健康检查接口"""
    log("=" * 60)
    log("开始测试健康检查接口")
    add_report_line("\n## 零、健康检查\n")
    
    try:
        response = requests.get(f"http://localhost:8080/ping", timeout=5)
        if response.status_code == 200:
            log("✓ 服务器健康检查通过", "SUCCESS")
            add_report_line(f"**服务器状态**: ✅ 正常运行\n")
            add_report_line(f"**响应**: {response.json()}\n")
            return True
        else:
            log("✗ 服务器健康检查失败", "ERROR")
            add_report_line(f"**服务器状态**: ❌ 异常 (状态码: {response.status_code})\n")
            return False
    except Exception as e:
        log(f"✗ 无法连接到服务器: {str(e)}", "ERROR")
        add_report_line(f"**服务器状态**: ❌ 无法连接 - {str(e)}\n")
        return False


def generate_report():
    """生成测试报告"""
    log("=" * 60)
    log("生成测试报告")
    
    # 报告头部
    report_header = f"""# 扑克记分系统后端接口测试报告

## 测试信息

- **测试时间**: {datetime.now().strftime("%Y-%m-%d %H:%M:%S")}
- **测试环境**: {BASE_URL}
- **测试工具**: Python + requests
- **测试人员**: AI Assistant

## 测试统计

- **总测试数**: {test_stats['total']}
- **通过数**: {test_stats['passed']} ✅
- **失败数**: {test_stats['failed']} ❌
- **通过率**: {(test_stats['passed'] / test_stats['total'] * 100) if test_stats['total'] > 0 else 0:.2f}%

"""
    
    if test_stats["errors"]:
        report_header += "\n## 失败的测试\n\n"
        for i, error in enumerate(test_stats["errors"], 1):
            report_header += f"{i}. {error}\n"
    
    report_header += "\n---\n"
    
    # 组合完整报告
    full_report = report_header + "\n".join(report_lines)
    
    # 测试数据摘要
    report_footer = f"""

---

## 测试数据摘要

### 创建的测试用户

"""
    
    for i, user in enumerate(test_data["users"], 1):
        report_footer += f"{i}. 用户ID: {user['id']}, 昵称: {user['nickname']}, 手机号: {user['phone']}\n"
    
    report_footer += "\n### 创建的测试房间\n\n"
    
    for i, room in enumerate(test_data["rooms"], 1):
        report_footer += f"{i}. 房间ID: {room['id']}, 房间号: {room['code']}, 类型: {room['type']}\n"
    
    report_footer += f"""

---

## 测试说明

1. 本测试脚本自动测试了所有主要的API接口
2. 测试按照接口文档的顺序进行,包括正常流程和异常流程
3. WebSocket接口由于其实时性特点,未包含在本自动化测试中,建议手动测试
4. 管理员接口测试需要手动修改数据库中用户的role字段为'admin'
5. 部分测试可能因为数据依赖而失败,建议在干净的测试环境中运行

## 建议

- 定期运行此测试脚本以确保接口的稳定性
- 在修改代码后运行测试以验证功能
- 可以根据实际需求扩展更多测试用例
- 建议配合集成测试和压力测试使用

---

**测试完成时间**: {datetime.now().strftime("%Y-%m-%d %H:%M:%S")}
"""
    
    full_report += report_footer
    
    # 保存报告
    try:
        with open(TEST_REPORT_FILE, "w", encoding="utf-8") as f:
            f.write(full_report)
        log(f"✓ 测试报告已保存到: {TEST_REPORT_FILE}", "SUCCESS")
    except Exception as e:
        log(f"✗ 保存测试报告失败: {str(e)}", "ERROR")


def main():
    """主函数"""
    print("=" * 60)
    print("扑克记分系统后端接口测试")
    print("=" * 60)
    
    # 检查服务器是否运行
    if not test_health_check():
        log("服务器未运行,请先启动后端服务!", "ERROR")
        sys.exit(1)
    
    # 执行测试
    test_auth_apis()
    test_room_apis()
    test_operation_apis()
    test_niuniu_operations()
    test_settlement_apis()
    test_record_apis()
    test_admin_apis()
    
    # 生成报告
    generate_report()
    
    # 输出总结
    print("\n" + "=" * 60)
    print("测试完成!")
    print(f"总测试数: {test_stats['total']}")
    print(f"通过: {test_stats['passed']} ✅")
    print(f"失败: {test_stats['failed']} ❌")
    print(f"通过率: {(test_stats['passed'] / test_stats['total'] * 100) if test_stats['total'] > 0 else 0:.2f}%")
    print(f"测试报告: {TEST_REPORT_FILE}")
    print("=" * 60)
    
    # 如果有失败的测试,返回非0退出码
    if test_stats["failed"] > 0:
        sys.exit(1)


if __name__ == "__main__":
    main()

