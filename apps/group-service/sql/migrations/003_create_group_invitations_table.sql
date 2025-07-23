-- 创建群邀请表
CREATE TABLE IF NOT EXISTS group_invitations (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL,
    inviter_id BIGINT NOT NULL,
    invitee_id BIGINT NOT NULL,
    status VARCHAR(20) DEFAULT 'pending',
    message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    expired_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT fk_group_invitations_group_id FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_group_invitations_group_id ON group_invitations(group_id);
CREATE INDEX IF NOT EXISTS idx_group_invitations_inviter_id ON group_invitations(inviter_id);
CREATE INDEX IF NOT EXISTS idx_group_invitations_invitee_id ON group_invitations(invitee_id);
CREATE INDEX IF NOT EXISTS idx_group_invitations_status ON group_invitations(status);
CREATE INDEX IF NOT EXISTS idx_group_invitations_created_at ON group_invitations(created_at);
CREATE INDEX IF NOT EXISTS idx_group_invitations_expired_at ON group_invitations(expired_at);

-- 创建唯一约束，确保同一个用户对同一个群组只能有一个待处理的邀请
CREATE UNIQUE INDEX IF NOT EXISTS idx_group_invitations_unique_pending 
ON group_invitations(group_id, invitee_id) 
WHERE status = 'pending';

-- 添加注释
COMMENT ON TABLE group_invitations IS '群邀请表';
COMMENT ON COLUMN group_invitations.id IS '邀请记录ID';
COMMENT ON COLUMN group_invitations.group_id IS '群组ID';
COMMENT ON COLUMN group_invitations.inviter_id IS '邀请人用户ID';
COMMENT ON COLUMN group_invitations.invitee_id IS '被邀请人用户ID';
COMMENT ON COLUMN group_invitations.status IS '邀请状态: pending(待处理), accepted(已接受), rejected(已拒绝), expired(已过期)';
COMMENT ON COLUMN group_invitations.message IS '邀请消息';
COMMENT ON COLUMN group_invitations.created_at IS '创建时间';
COMMENT ON COLUMN group_invitations.updated_at IS '更新时间';
COMMENT ON COLUMN group_invitations.expired_at IS '过期时间';
