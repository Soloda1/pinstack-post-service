
CREATE TABLE IF NOT EXISTS posts (
         id          SERIAL PRIMARY KEY,
         author_id   UUID        NOT NULL,
         title       TEXT        NOT NULL,
         content     TEXT,
         created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
         updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);


CREATE TABLE IF NOT EXISTS post_media (
      id         SERIAL        PRIMARY KEY,
      post_id    INT            NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
      url        TEXT           NOT NULL,
      type       TEXT           NOT NULL CHECK (type IN ('photo','video')),
      position   SMALLINT       NOT NULL CHECK (position BETWEEN 1 AND 9),
      created_at TIMESTAMPTZ    NOT NULL DEFAULT now()
);


CREATE TABLE IF NOT EXISTS tags (
    id    SERIAL PRIMARY KEY,
    name  TEXT   NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS post_tags (
     post_id INT NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
     tag_id  INT NOT NULL REFERENCES tags(id)  ON DELETE CASCADE,
     PRIMARY KEY (post_id, tag_id)
);


CREATE INDEX IF NOT EXISTS idx_post_media_post_id_position
    ON post_media(post_id, position);

CREATE INDEX IF NOT EXISTS idx_post_tags_tag_id
    ON post_tags(tag_id);

CREATE INDEX IF NOT EXISTS idx_posts_author_id
    ON posts(author_id);
