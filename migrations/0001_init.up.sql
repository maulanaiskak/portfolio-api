-- 0001_init: admin user, CV tables, and content tables (trips, milestones, posts).

-- Keeps updated_at current on every UPDATE.
CREATE OR REPLACE FUNCTION set_updated_at() RETURNS trigger AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ---------------------------------------------------------------------------
-- Admin
-- ---------------------------------------------------------------------------

CREATE TABLE admin_users (
    id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    email         TEXT        NOT NULL CHECK (email = lower(email) AND length(email) BETWEEN 3 AND 254),
    password_hash TEXT        NOT NULL CHECK (password_hash LIKE '$2%'),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT admin_users_email_key UNIQUE (email)
);

CREATE TRIGGER admin_users_set_updated_at BEFORE UPDATE ON admin_users
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ---------------------------------------------------------------------------
-- CV (mirrors maulanaiskak-nextjs/src/types/index.ts; nested arrays as JSONB)
-- ---------------------------------------------------------------------------

CREATE TABLE cv_experiences (
    id                        BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    company                   TEXT        NOT NULL CHECK (company <> ''),
    location                  TEXT        NOT NULL DEFAULT '',
    position                  TEXT        NOT NULL CHECK (position <> ''),
    period                    TEXT        NOT NULL DEFAULT '',
    -- [{ "period": string, "description": string }]
    responsibilities          JSONB       NOT NULL DEFAULT '[]' CHECK (jsonb_typeof(responsibilities) = 'array'),
    projects_and_achievements JSONB       NOT NULL DEFAULT '[]' CHECK (jsonb_typeof(projects_and_achievements) = 'array'),
    tech                      JSONB       NOT NULL DEFAULT '[]' CHECK (jsonb_typeof(tech) = 'array'),
    sort_order                INT         NOT NULL DEFAULT 0,
    created_at                TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX cv_experiences_sort_idx ON cv_experiences (sort_order, id);
CREATE TRIGGER cv_experiences_set_updated_at BEFORE UPDATE ON cv_experiences
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE cv_projects (
    id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    title        TEXT        NOT NULL CHECK (title <> ''),
    description  TEXT        NOT NULL DEFAULT '',
    technologies JSONB       NOT NULL DEFAULT '[]' CHECK (jsonb_typeof(technologies) = 'array'),
    image        TEXT        NOT NULL DEFAULT '',
    highlights   JSONB       NOT NULL DEFAULT '[]' CHECK (jsonb_typeof(highlights) = 'array'),
    sort_order   INT         NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX cv_projects_sort_idx ON cv_projects (sort_order, id);
CREATE TRIGGER cv_projects_set_updated_at BEFORE UPDATE ON cv_projects
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE cv_education (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    institution TEXT        NOT NULL CHECK (institution <> ''),
    location    TEXT        NOT NULL DEFAULT '',
    degree      TEXT        NOT NULL DEFAULT '',
    period      TEXT        NOT NULL DEFAULT '',
    gpa         TEXT        NOT NULL DEFAULT '',
    thesis      TEXT        NOT NULL DEFAULT '',
    thesis_link TEXT        NOT NULL DEFAULT '',
    sort_order  INT         NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX cv_education_sort_idx ON cv_education (sort_order, id);
CREATE TRIGGER cv_education_set_updated_at BEFORE UPDATE ON cv_education
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE cv_skill_categories (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    category   TEXT        NOT NULL CHECK (category <> ''),
    sort_order INT         NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT cv_skill_categories_category_key UNIQUE (category)
);

CREATE INDEX cv_skill_categories_sort_idx ON cv_skill_categories (sort_order, id);
CREATE TRIGGER cv_skill_categories_set_updated_at BEFORE UPDATE ON cv_skill_categories
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE cv_skill_items (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    category_id BIGINT      NOT NULL REFERENCES cv_skill_categories (id) ON DELETE CASCADE,
    name        TEXT        NOT NULL CHECK (name <> ''),
    level       SMALLINT    NOT NULL CHECK (level BETWEEN 0 AND 100),
    note        TEXT,
    link        TEXT CHECK (link IS NULL OR link ~ '^https?://'),
    sort_order  INT         NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX cv_skill_items_category_sort_idx ON cv_skill_items (category_id, sort_order, id);
CREATE TRIGGER cv_skill_items_set_updated_at BEFORE UPDATE ON cv_skill_items
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ---------------------------------------------------------------------------
-- Content: trips, career milestones, blog posts
-- ---------------------------------------------------------------------------
-- Slugs are URL-safe lowercase kebab-case. A published row must carry a
-- published_at so public list ordering is always defined.

CREATE TABLE trips (
    id              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    slug            TEXT        NOT NULL CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$'),
    title           TEXT        NOT NULL CHECK (title <> ''),
    location        TEXT        NOT NULL DEFAULT '',
    country_code    CHAR(2)     CHECK (country_code IS NULL OR country_code ~ '^[A-Z]{2}$'),
    start_date      DATE,
    end_date        DATE,
    summary         TEXT        NOT NULL DEFAULT '',
    body_markdown   TEXT        NOT NULL DEFAULT '',
    cover_image_url TEXT CHECK (cover_image_url IS NULL OR cover_image_url ~ '^https?://'),
    status          TEXT        NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'published')),
    published_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT trips_slug_key UNIQUE (slug),
    CONSTRAINT trips_dates_check CHECK (end_date IS NULL OR start_date IS NULL OR end_date >= start_date),
    CONSTRAINT trips_published_at_check CHECK (status <> 'published' OR published_at IS NOT NULL)
);

CREATE INDEX trips_status_published_idx ON trips (status, published_at DESC);
CREATE TRIGGER trips_set_updated_at BEFORE UPDATE ON trips
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE trip_photos (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    trip_id    BIGINT      NOT NULL REFERENCES trips (id) ON DELETE CASCADE,
    url        TEXT        NOT NULL CHECK (url ~ '^https?://'),
    caption    TEXT        NOT NULL DEFAULT '',
    sort_order INT         NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX trip_photos_trip_sort_idx ON trip_photos (trip_id, sort_order, id);
CREATE TRIGGER trip_photos_set_updated_at BEFORE UPDATE ON trip_photos
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE career_milestones (
    id              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    slug            TEXT        NOT NULL CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$'),
    title           TEXT        NOT NULL CHECK (title <> ''),
    company         TEXT        NOT NULL DEFAULT '',
    role            TEXT        NOT NULL DEFAULT '',
    milestone_date  DATE,
    summary         TEXT        NOT NULL DEFAULT '',
    body_markdown   TEXT        NOT NULL DEFAULT '',
    cover_image_url TEXT CHECK (cover_image_url IS NULL OR cover_image_url ~ '^https?://'),
    status          TEXT        NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'published')),
    published_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT career_milestones_slug_key UNIQUE (slug),
    CONSTRAINT career_milestones_published_at_check CHECK (status <> 'published' OR published_at IS NOT NULL)
);

CREATE INDEX career_milestones_status_published_idx ON career_milestones (status, published_at DESC);
CREATE TRIGGER career_milestones_set_updated_at BEFORE UPDATE ON career_milestones
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE blog_posts (
    id              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    slug            TEXT        NOT NULL CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$'),
    title           TEXT        NOT NULL CHECK (title <> ''),
    summary         TEXT        NOT NULL DEFAULT '',
    body_markdown   TEXT        NOT NULL DEFAULT '',
    cover_image_url TEXT CHECK (cover_image_url IS NULL OR cover_image_url ~ '^https?://'),
    -- ["go", "travel", ...]
    tags            JSONB       NOT NULL DEFAULT '[]' CHECK (jsonb_typeof(tags) = 'array'),
    status          TEXT        NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'published')),
    published_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT blog_posts_slug_key UNIQUE (slug),
    CONSTRAINT blog_posts_published_at_check CHECK (status <> 'published' OR published_at IS NOT NULL)
);

CREATE INDEX blog_posts_status_published_idx ON blog_posts (status, published_at DESC);
-- Serves GET /api/v1/posts?tag=... via `tags @> '["tag"]'`.
CREATE INDEX blog_posts_tags_idx ON blog_posts USING GIN (tags jsonb_path_ops);
CREATE TRIGGER blog_posts_set_updated_at BEFORE UPDATE ON blog_posts
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
