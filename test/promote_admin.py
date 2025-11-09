#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
提升用户为管理员的脚本
"""

import sqlite3
import sys

def promote_user_to_admin(phone):
    """将指定手机号的用户提升为管理员"""
    try:
        # 连接数据库
        conn = sqlite3.connect('../backend/database.db')
        cursor = conn.cursor()
        
        # 更新用户角色
        cursor.execute("UPDATE users SET role = 'admin' WHERE phone = ?", (phone,))
        
        if cursor.rowcount == 0:
            print(f"❌ 未找到手机号为 {phone} 的用户")
            return False
        
        conn.commit()
        conn.close()
        
        print(f"✅ 成功将手机号 {phone} 的用户提升为管理员")
        return True
        
    except Exception as e:
        print(f"❌ 操作失败: {str(e)}")
        return False


if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("使用方法: python3 promote_admin.py <phone>")
        print("示例: python3 promote_admin.py 13800138000")
        sys.exit(1)
    
    phone = sys.argv[1]
    promote_user_to_admin(phone)

