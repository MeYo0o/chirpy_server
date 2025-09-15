-- +goose Up
CREATE TABLE users(
  id UUID PRIMARY KEY,
  email TEXT UNIQUE NOT NULL,
  hashed_password TEXT NOT NULL DEFAULT 'unset',
  is_chirpy_red BOOL NOT NULL DEFAULT FALSE,
  created_at TIMESTAMP NOT NULL,
  updated_at TIMESTAMP NOT NULL
);
-- +goose Down
DROP TABLE users;