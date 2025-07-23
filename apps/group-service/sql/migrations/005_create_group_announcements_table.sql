-- 创建群公告表
CREATE TABLE IF NOT EXISTS group_announcements (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL,
    author_id BIGINT NOT NULL,
    title VARCHAR(200),
    content TEXT NOT NULL,
    is_pinned BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_group_announcements_group_id FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_group_announcements_group_id ON group_announcements(group_id);
CREATE INDEX IF NOT EXISTS idx_group_announcements_author_id ON group_announcements(author_id);
CREATE INDEX IF NOT EXISTS idx_group_announcements_is_pinned ON group_announcements(is_pinned);
CREATE INDEX IF NOT EXISTS idx_group_announcements_created_at ON group_announcements(created_at);

-- 添加注释
COMMENT ON TABLE group_announcements IS '群公告表';
COMMENT ON COLUMN group_announcements.id IS '公告ID';
COMMENT ON COLUMN group_announcements.group_id IS '群组ID';
COMMENT ON COLUMN group_announcements.author_id IS '发布人用户ID';
COMMENT ON COLUMN group_announcements.title IS '公告标题';
COMMENT ON COLUMN group_announcements.content IS '公告内容';
COMMENT ON COLUMN group_announcements.is_pinned IS '是否置顶';
COMMENT ON COLUMN group_announcements.created_at IS '创建时间';
COMMENT ON COLUMN group_announcements.updated_at IS '更新时间';
