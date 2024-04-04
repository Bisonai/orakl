CREATE TABLE IF NOT EXISTS provider_urls (
    id SERIAL PRIMARY KEY,
    chain TEXT NOT NULL,
    chain_id INT NOT NULL,
    url TEXT NOT NULL,
    priority INT
)