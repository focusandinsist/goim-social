-- 创建加群申请表
CREATE TABLE IF NOT EXISTS group_join_requests (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    reason TEXT,
    status VARCHAR(20) DEFAULT 'pending',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    processed_by BIGINT,
    processed_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT fk_group_join_requests_group_id FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_group_join_requests_group_id ON group_join_requests(group_id);
CREATE INDEX IF NOT EXISTS idx_group_join_requests_user_id ON group_join_requests(user_id);
CREATE INDEX IF NOT EXISTS idx_group_join_requests_status ON group_join_requests(status);
CREATE INDEX IF NOT EXISTS idx_group_join_requests_created_at ON group_join_requests(created_at);
CREATE INDEX IF NOT EXISTS idx_group_join_requests_processed_by ON group_join_requests(processed_by);

-- 创建唯一约束，确保同一个用户对同一个群组只能有一个待处理的申请
CREATE UNIQUE INDEX IF NOT EXISTS idx_group_join_requests_unique_pending 
ON group_join_requests(group_id, user_id) 
WHERE status = 'pending';

-- 添加注释
COMMENT ON TABLE group_join_requests IS '加群申请表';
COMMENT ON COLUMN group_join_requests.id IS '申请记录ID';
COMMENT ON COLUMN group_join_requests.group_id IS '群组ID';
COMMENT ON COLUMN group_join_requests.user_id IS '申请人用户ID';
COMMENT ON COLUMN group_join_requests.reason IS '申请理由';
COMMENT ON COLUMN group_join_requests.status IS '申请状态: pending(待处理), approved(已通过), rejected(已拒绝)';
COMMENT ON COLUMN group_join_requests.created_at IS '创建时间';
COMMENT ON COLUMN group_join_requests.updated_at IS '更新时间';
COMMENT ON COLUMN group_join_requests.processed_by IS '处理人用户ID';
COMMENT ON COLUMN group_join_requests.processed_at IS '处理时间';
