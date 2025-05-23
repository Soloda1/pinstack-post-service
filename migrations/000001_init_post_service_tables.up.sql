
CREATE TABLE IF NOT EXISTS posts (
         id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
         author_id   bigint        NOT NULL,
         title       TEXT        NOT NULL,
         content     TEXT,
         created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
         updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);


CREATE TABLE IF NOT EXISTS post_media (
      id         bigint GENERATED ALWAYS AS IDENTITY        PRIMARY KEY,
      post_id    bigint            NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
      url        TEXT           NOT NULL,
      type       TEXT           NOT NULL CHECK (type IN ('image','video')),
      position   SMALLINT       NOT NULL CHECK (position BETWEEN 1 AND 9),
      created_at TIMESTAMPTZ    NOT NULL DEFAULT now()
);


CREATE TABLE IF NOT EXISTS tags (
    id    bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name  TEXT   NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS posts_tags (
     post_id bigint NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
     tag_id  bigint NOT NULL REFERENCES tags(id)  ON DELETE CASCADE,
     PRIMARY KEY (post_id, tag_id)
);


CREATE INDEX IF NOT EXISTS idx_post_media_post_id_position
    ON post_media(post_id, position);

CREATE INDEX IF NOT EXISTS idx_posts_tags_tag_id
    ON posts_tags(tag_id);

CREATE INDEX IF NOT EXISTS idx_posts_author_id
    ON posts(author_id);

CREATE INDEX idx_tags_name ON tags(name);
