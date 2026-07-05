ALTER TABLE posts
    ADD COLUMN width INT,
    ADD COLUMN height INT,
    ADD COLUMN aspect_ratio FLOAT,
    ADD COLUMN thumbnail_url VARCHAR,
    ADD COLUMN blur_hash VARCHAR,
    ADD COLUMN duration INT,
    ADD COLUMN file_size INT,
    ADD COLUMN mime_type VARCHAR;

ALTER TABLE reels
    ADD COLUMN width INT,
    ADD COLUMN height INT,
    ADD COLUMN aspect_ratio FLOAT,
    ADD COLUMN thumbnail_url VARCHAR,
    ADD COLUMN blur_hash VARCHAR,
    ADD COLUMN duration INT,
    ADD COLUMN file_size INT,
    ADD COLUMN mime_type VARCHAR;

ALTER TABLE stories
    ADD COLUMN width INT,
    ADD COLUMN height INT,
    ADD COLUMN aspect_ratio FLOAT,
    ADD COLUMN blur_hash VARCHAR,
    ADD COLUMN duration INT,
    ADD COLUMN file_size INT,
    ADD COLUMN mime_type VARCHAR;
