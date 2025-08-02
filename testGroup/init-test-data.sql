-- 群聊测试数据初始化脚本
-- 在PostgreSQL数据库中执行此脚本来准备测试数据

-- 清理现有测试数据 (可选)
-- DELETE FROM group_members WHERE group_id IN (1001, 1002, 1003);
-- DELETE FROM groups WHERE id IN (1001, 1002, 1003);

-- 创建测试群组
INSERT INTO groups (id, name, description, owner_id, member_count, max_members, is_public, announcement, created_at, updated_at) VALUES
(1001, '技术讨论群', '用于讨论技术问题和分享经验', 1001, 4, 500, true, '欢迎大家积极讨论技术问题！', NOW(), NOW()),
(1002, '项目协作群', '项目开发协作和进度同步', 1002, 3, 100, false, '请及时汇报项目进度', NOW(), NOW()),
(1003, '测试群聊', '用于测试群聊功能', 1001, 5, 50, true, '这是一个测试群组', NOW(), NOW())
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    member_count = EXCLUDED.member_count,
    updated_at = NOW();

-- 添加群成员
INSERT INTO group_members (user_id, group_id, role, nickname, joined_at) VALUES
-- 技术讨论群 (1001) 成员
(1001, 1001, 'owner', '群主张三', NOW()),
(1002, 1001, 'admin', '管理员李四', NOW()),
(1003, 1001, 'member', '开发王五', NOW()),
(1004, 1001, 'member', '测试赵六', NOW()),

-- 项目协作群 (1002) 成员
(1002, 1002, 'owner', '项目经理李四', NOW()),
(1001, 1002, 'admin', '技术负责人张三', NOW()),
(1005, 1002, 'member', '前端小明', NOW()),

-- 测试群聊 (1003) 成员
(1001, 1003, 'owner', '测试用户1', NOW()),
(1002, 1003, 'member', '测试用户2', NOW()),
(1003, 1003, 'member', '测试用户3', NOW()),
(1004, 1003, 'member', '测试用户4', NOW()),
(1005, 1003, 'member', '测试用户5', NOW())
ON CONFLICT (user_id, group_id) DO UPDATE SET
    nickname = EXCLUDED.nickname,
    joined_at = EXCLUDED.joined_at;

-- 查询验证数据
SELECT 
    g.id as group_id,
    g.name as group_name,
    g.member_count,
    g.is_public,
    COUNT(gm.user_id) as actual_member_count
FROM groups g
LEFT JOIN group_members gm ON g.id = gm.group_id
WHERE g.id IN (1001, 1002, 1003)
GROUP BY g.id, g.name, g.member_count, g.is_public
ORDER BY g.id;

-- 显示群成员详情
SELECT 
    gm.group_id,
    g.name as group_name,
    gm.user_id,
    gm.role,
    gm.nickname,
    gm.joined_at
FROM group_members gm
JOIN groups g ON gm.group_id = g.id
WHERE gm.group_id IN (1001, 1002, 1003)
ORDER BY gm.group_id, gm.user_id;
