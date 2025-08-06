-- 社交服务数据库迁移脚本
-- 用于从 friend-service 和 group-service 迁移数据到 social-service

-- 创建新的社交服务数据库
CREATE DATABASE IF NOT EXISTS goim_social;
USE goim_social;

-- ============ 好友关系相关表 ============

-- 好友关系表
CREATE TABLE IF NOT EXISTS friends (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    friend_id BIGINT NOT NULL,
    remark VARCHAR(100) DEFAULT '',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- 索引
    INDEX idx_user_id (user_id),
    INDEX idx_friend_id (friend_id),
    
    -- 唯一约束
    UNIQUE KEY uk_user_friend (user_id, friend_id)
);

-- 好友申请表
CREATE TABLE IF NOT EXISTS friend_applies (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL COMMENT '被申请人',
    applicant_id BIGINT NOT NULL COMMENT '申请人',
    remark TEXT COMMENT '申请备注',
    status VARCHAR(20) DEFAULT 'pending' COMMENT '状态: pending/accepted/rejected',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    agree_time TIMESTAMP NULL COMMENT '同意时间',
    reject_time TIMESTAMP NULL COMMENT '拒绝时间',
    agree_remark VARCHAR(100) DEFAULT '' COMMENT '同意时备注',
    reject_reason VARCHAR(200) DEFAULT '' COMMENT '拒绝原因',
    
    -- 索引
    INDEX idx_user_id (user_id),
    INDEX idx_applicant_id (applicant_id),
    INDEX idx_status (status),
    INDEX idx_created_at (created_at),
    
    -- 唯一约束
    UNIQUE KEY uk_user_applicant (user_id, applicant_id)
);

-- ============ 群组相关表 ============

-- 群组表
CREATE TABLE IF NOT EXISTS groups (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL COMMENT '群组名称',
    description TEXT COMMENT '群组描述',
    avatar VARCHAR(500) DEFAULT '' COMMENT '群组头像',
    owner_id BIGINT NOT NULL COMMENT '群主ID',
    member_count INTEGER DEFAULT 1 COMMENT '成员数量',
    max_members INTEGER DEFAULT 500 COMMENT '最大成员数',
    is_public BOOLEAN DEFAULT TRUE COMMENT '是否公开',
    announcement TEXT COMMENT '群公告',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    -- 索引
    INDEX idx_name (name),
    INDEX idx_owner_id (owner_id),
    INDEX idx_is_public (is_public),
    INDEX idx_created_at (created_at)
);

-- 群成员表
CREATE TABLE IF NOT EXISTS group_members (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL COMMENT '用户ID',
    group_id BIGINT NOT NULL COMMENT '群组ID',
    role VARCHAR(20) DEFAULT 'member' COMMENT '角色: owner/admin/member',
    nickname VARCHAR(100) DEFAULT '' COMMENT '群昵称',
    joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '加入时间',
    
    -- 索引
    INDEX idx_user_id (user_id),
    INDEX idx_group_id (group_id),
    INDEX idx_role (role),
    INDEX idx_joined_at (joined_at),
    
    -- 唯一约束
    UNIQUE KEY uk_user_group (user_id, group_id)
);

-- 群邀请表
CREATE TABLE IF NOT EXISTS group_invitations (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL COMMENT '群组ID',
    inviter_id BIGINT NOT NULL COMMENT '邀请人ID',
    invitee_id BIGINT NOT NULL COMMENT '被邀请人ID',
    status VARCHAR(20) DEFAULT 'pending' COMMENT '状态: pending/accepted/rejected/expired',
    message TEXT COMMENT '邀请消息',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    expired_at TIMESTAMP NOT NULL COMMENT '过期时间',
    
    -- 索引
    INDEX idx_group_id (group_id),
    INDEX idx_inviter_id (inviter_id),
    INDEX idx_invitee_id (invitee_id),
    INDEX idx_status (status),
    INDEX idx_expired_at (expired_at)
);

-- 加群申请表
CREATE TABLE IF NOT EXISTS group_join_requests (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL COMMENT '群组ID',
    user_id BIGINT NOT NULL COMMENT '申请人ID',
    status VARCHAR(20) DEFAULT 'pending' COMMENT '状态: pending/approved/rejected',
    reason TEXT COMMENT '申请理由',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    -- 索引
    INDEX idx_group_id (group_id),
    INDEX idx_user_id (user_id),
    INDEX idx_status (status),
    INDEX idx_created_at (created_at),
    
    -- 唯一约束
    UNIQUE KEY uk_group_user (group_id, user_id)
);

-- ============ 数据迁移脚本 ============

-- 如果存在原有的 friend-service 数据库，迁移好友数据
-- INSERT INTO goim_social.friends (user_id, friend_id, remark, created_at)
-- SELECT user_id, friend_id, remark, created_at 
-- FROM friend_service.friends;

-- INSERT INTO goim_social.friend_applies (user_id, applicant_id, remark, status, created_at, updated_at)
-- SELECT user_id, applicant_id, remark, status, created_at, updated_at
-- FROM friend_service.friend_applies;

-- 如果存在原有的 group-service 数据库，迁移群组数据
-- INSERT INTO goim_social.groups (id, name, description, avatar, owner_id, member_count, max_members, is_public, announcement, created_at, updated_at)
-- SELECT id, name, description, avatar, owner_id, member_count, max_members, is_public, announcement, created_at, updated_at
-- FROM group_service.groups;

-- INSERT INTO goim_social.group_members (user_id, group_id, role, nickname, joined_at)
-- SELECT user_id, group_id, role, nickname, joined_at
-- FROM group_service.group_members;

-- ============ 创建视图和存储过程 ============

-- 用户社交信息汇总视图
CREATE OR REPLACE VIEW user_social_summary AS
SELECT 
    u.user_id,
    COALESCE(f.friend_count, 0) as friend_count,
    COALESCE(g.group_count, 0) as group_count
FROM (
    SELECT DISTINCT user_id FROM friends
    UNION
    SELECT DISTINCT user_id FROM group_members
) u
LEFT JOIN (
    SELECT user_id, COUNT(*) as friend_count
    FROM friends
    GROUP BY user_id
) f ON u.user_id = f.user_id
LEFT JOIN (
    SELECT user_id, COUNT(*) as group_count
    FROM group_members
    GROUP BY user_id
) g ON u.user_id = g.user_id;

-- 获取用户好友ID列表的存储过程
DELIMITER //
CREATE PROCEDURE GetUserFriendIDs(IN p_user_id BIGINT)
BEGIN
    SELECT friend_id FROM friends WHERE user_id = p_user_id;
END //
DELIMITER ;

-- 获取群组成员ID列表的存储过程
DELIMITER //
CREATE PROCEDURE GetGroupMemberIDs(IN p_group_id BIGINT)
BEGIN
    SELECT user_id FROM group_members WHERE group_id = p_group_id;
END //
DELIMITER ;

-- 验证好友关系的存储过程
DELIMITER //
CREATE PROCEDURE ValidateFriendship(IN p_user_id BIGINT, IN p_friend_id BIGINT, OUT p_is_friend BOOLEAN)
BEGIN
    DECLARE friend_count INT DEFAULT 0;
    
    SELECT COUNT(*) INTO friend_count
    FROM friends
    WHERE user_id = p_user_id AND friend_id = p_friend_id;
    
    SET p_is_friend = (friend_count > 0);
END //
DELIMITER ;

-- 验证群成员关系的存储过程
DELIMITER //
CREATE PROCEDURE ValidateGroupMembership(IN p_user_id BIGINT, IN p_group_id BIGINT, OUT p_is_member BOOLEAN)
BEGIN
    DECLARE member_count INT DEFAULT 0;
    
    SELECT COUNT(*) INTO member_count
    FROM group_members
    WHERE user_id = p_user_id AND group_id = p_group_id;
    
    SET p_is_member = (member_count > 0);
END //
DELIMITER ;

-- ============ 初始化数据 ============

-- 插入一些测试数据（可选）
-- INSERT INTO groups (name, description, owner_id, is_public) VALUES
-- ('技术交流群', '讨论技术问题的群组', 1, TRUE),
-- ('产品讨论群', '产品相关讨论', 2, TRUE),
-- ('内部工作群', '内部工作交流', 1, FALSE);

-- INSERT INTO group_members (user_id, group_id, role) VALUES
-- (1, 1, 'owner'),
-- (2, 1, 'member'),
-- (3, 1, 'member'),
-- (2, 2, 'owner'),
-- (1, 2, 'member');

-- INSERT INTO friends (user_id, friend_id, remark) VALUES
-- (1, 2, '同事'),
-- (2, 1, '同事'),
-- (1, 3, '朋友'),
-- (3, 1, '朋友'),
-- (2, 3, '同学'),
-- (3, 2, '同学');

COMMIT;
