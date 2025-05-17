DROP INDEX IF EXISTS idx_post_media_post_id_position;
DROP INDEX IF EXISTS idx_post_tags_tag_id;
DROP INDEX IF EXISTS idx_posts_author_id;

DROP TABLE IF EXISTS post_tags;
DROP TABLE IF EXISTS post_media;
DROP TABLE IF EXISTS tags;
DROP TABLE IF EXISTS posts;