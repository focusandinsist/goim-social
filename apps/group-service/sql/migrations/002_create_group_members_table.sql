-- 创建群成员表
CREATE TABLE IF NOT EXISTS group_members (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    group_id BIGINT NOT NULL,
    role VARCHAR(20) DEFAULT 'member',
    nickname VARCHAR(100),
    joined_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_group_members_group_id FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_group_members_user_id ON group_members(user_id);
CREATE INDEX IF NOT EXISTS idx_group_members_group_id ON group_members(group_id);
CREATE INDEX IF NOT EXISTS idx_group_members_role ON group_members(role);
CREATE INDEX IF NOT EXISTS idx_group_members_joined_at ON group_members(joined_at);

-- 创建唯一约束，确保用户在同一个群组中只能有一个记录
CREATE UNIQUE INDEX IF NOT EXISTS idx_group_members_unique ON group_members(user_id, group_id);

-- 添加注释
COMMENT ON TABLE group_members IS '群成员表';
COMMENT ON COLUMN group_members.id IS '成员记录ID';
COMMENT ON COLUMN group_members.user_id IS '用户ID';
COMMENT ON COLUMN group_members.group_id IS '群组ID';
COMMENT ON COLUMN group_members.role IS '成员角色: owner(群主), admin(管理员), member(普通成员)';
COMMENT ON COLUMN group_members.nickname IS '群内昵称';
COMMENT ON COLUMN group_members.joined_at IS '加入时间';
