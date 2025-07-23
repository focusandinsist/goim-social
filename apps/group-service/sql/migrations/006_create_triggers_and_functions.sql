-- 创建更新时间触发器函数
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- 为groups表创建更新时间触发器
DROP TRIGGER IF EXISTS update_groups_updated_at ON groups;
CREATE TRIGGER update_groups_updated_at
    BEFORE UPDATE ON groups
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 为group_invitations表创建更新时间触发器
DROP TRIGGER IF EXISTS update_group_invitations_updated_at ON group_invitations;
CREATE TRIGGER update_group_invitations_updated_at
    BEFORE UPDATE ON group_invitations
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 为group_join_requests表创建更新时间触发器
DROP TRIGGER IF EXISTS update_group_join_requests_updated_at ON group_join_requests;
CREATE TRIGGER update_group_join_requests_updated_at
    BEFORE UPDATE ON group_join_requests
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 为group_announcements表创建更新时间触发器
DROP TRIGGER IF EXISTS update_group_announcements_updated_at ON group_announcements;
CREATE TRIGGER update_group_announcements_updated_at
    BEFORE UPDATE ON group_announcements
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 创建自动更新群成员数量的函数
CREATE OR REPLACE FUNCTION update_group_member_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        -- 成员加入时增加计数
        UPDATE groups 
        SET member_count = member_count + 1 
        WHERE id = NEW.group_id;
        RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
        -- 成员离开时减少计数
        UPDATE groups 
        SET member_count = member_count - 1 
        WHERE id = OLD.group_id;
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$$ language 'plpgsql';

-- 为group_members表创建成员数量更新触发器
DROP TRIGGER IF EXISTS update_member_count_on_insert ON group_members;
CREATE TRIGGER update_member_count_on_insert
    AFTER INSERT ON group_members
    FOR EACH ROW
    EXECUTE FUNCTION update_group_member_count();

DROP TRIGGER IF EXISTS update_member_count_on_delete ON group_members;
CREATE TRIGGER update_member_count_on_delete
    AFTER DELETE ON group_members
    FOR EACH ROW
    EXECUTE FUNCTION update_group_member_count();
