-- +goose Up
-- +goose StatementBegin
CREATE TABLE users (
       id UUID PRIMARY KEY,
       login VARCHAR(255) NOT NULL,
       password VARCHAR(255) NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP users;
-- +goose StatementEnd
