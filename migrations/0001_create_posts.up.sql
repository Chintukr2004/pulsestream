CREATE TABLE IF NOT EXISTS posts (
    id uuid PRIMARY KEY,
    username TEXT NOT NULL,
    content TEXT NOT NULL,  
    created_at TIMESTAMP NOT NULL
);