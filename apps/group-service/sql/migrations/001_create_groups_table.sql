-- 创建群组表
CREATE TABLE IF NOT EXISTS groups (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    avatar VARCHAR(500),
    owner_id BIGINT NOT NULL,
    member_count INTEGER DEFAULT 1,
    max_members INTEGER DEFAULT 500,
    is_public BOOLEAN DEFAULT true,
    announcement TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_groups_name ON groups(name);
CREATE INDEX IF NOT EXISTS idx_groups_owner_id ON groups(owner_id);
CREATE INDEX IF NOT EXISTS idx_groups_is_public ON groups(is_public);
CREATE INDEX IF NOT EXISTS idx_groups_created_at ON groups(created_at);

-- 添加注释
COMMENT ON TABLE groups IS '群组表';
COMMENT ON COLUMN groups.id IS '群组ID';
COMMENT ON COLUMN groups.name IS '群组名称';
COMMENT ON COLUMN groups.description IS '群组描述';
COMMENT ON COLUMN groups.avatar IS '群组头像URL';
COMMENT ON COLUMN groups.owner_id IS '群主用户ID';
COMMENT ON COLUMN groups.member_count IS '当前成员数量';
COMMENT ON COLUMN groups.max_members IS '最大成员数量';
COMMENT ON COLUMN groups.is_public IS '是否公开群组';
COMMENT ON COLUMN groups.announcement IS '群公告';
COMMENT ON COLUMN groups.created_at IS '创建时间';
COMMENT ON COLUMN groups.updated_at IS '更新时间';
