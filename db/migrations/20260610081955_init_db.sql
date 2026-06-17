-- migrate:up

CREATE TYPE USER_ROLE AS ENUM (
    'user',
    'admin'
);

CREATE TYPE GROUP_ROLE AS ENUM (
    'member',
    'maintainer',
    'owner'
);

CREATE TYPE OAUTH_STATUS AS ENUM (
    'pending',
    'approved',
    'denied'
);

CREATE TABLE users (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username       TEXT        NOT NULL UNIQUE,
    email          TEXT        NOT NULL UNIQUE,
    password_hash  TEXT        NOT NULL,
    role           USER_ROLE   NOT NULL DEFAULT 'user',
    blocked        BOOLEAN     NOT NULL DEFAULT false,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE groups (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT        NOT NULL UNIQUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE user_groups (
    user_id     UUID NOT NULL REFERENCES users(id)  ON DELETE CASCADE,
    group_id    UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    role        GROUP_ROLE NOT NULL DEFAULT 'member',
    joined_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, group_id)
);

CREATE TABLE sessions (
    token       TEXT PRIMARY KEY,
    user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX sessions_user_id_idx ON sessions(user_id);
CREATE INDEX sessions_expires_at_idx ON sessions(expires_at);

CREATE TABLE anonymous_users (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE device_authorizations (
      device_code   TEXT         PRIMARY KEY NOT NULL,
      user_code     TEXT         UNIQUE NOT NULL,
      status        oauth_status NOT NULL DEFAULT 'pending',
      user_id       UUID,
      session_token TEXT,
      expires_at    TIMESTAMPTZ  NOT NULL,
      created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),

      FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
      FOREIGN KEY (session_token) REFERENCES sessions(token) ON DELETE CASCADE
  );


-- migrate:down
DROP TABLE anonymous_users;
DROP TABLE user_groups;
DROP TABLE groups;
DROP TABLE users;
DROP TABLE sessions;
DROP TABLE device_authorizations;
DROP TYPE USER_ROLE;
DROP TYPE GROUP_ROLE;
DROP TYPE OAUTH_STATUS;
