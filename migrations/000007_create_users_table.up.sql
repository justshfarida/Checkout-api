CREATE TABLE users (
                       id SERIAL PRIMARY KEY,
                       username VARCHAR(50) NOT NULL UNIQUE,
                       password TEXT NOT NULL,
                       first_name VARCHAR(100),
                       last_name VARCHAR(100),
                       created_at TIMESTAMP NOT NULL DEFAULT NOW()
);