CREATE TABLE users(
    id SERIAL PRIMARY KEY,
    username VARCHAR(100) UNIQUE NOT NULL,
    access_level VARCHAR(20) CHECK (access_level IN ('admin', 'operator', 'viewer')),
    last_login TIMESTAMPTZ
);

CREATE TABLE computers(
    id SERIAL PRIMARY KEY,
    pc_name VARCHAR(255) UNIQUE NOT NULL,
    os VARCHAR(255) NOT NULL,
    arch VARCHAR(255) NOT NULL,
    os_version VARCHAR(255)
);

CREATE TABLE raw_events(
    id SERIAL PRIMARY KEY,
    username VARCHAR(255) REFERENCES users(username),
    pc_name VARCHAR(255) REFERENCES computers(pc_name),
    log_source VARCHAR(255) NOT NULL,
    event_timestamp TIMESTAMPTZ NOT NULL,
    raw_data TEXT
);

CREATE TABLE normalized_events(
    id SERIAL PRIMARY KEY,
    raw_event_id INTEGER REFERENCES raw_events(id),
    pc_name VARCHAR(255) REFERENCES computers(pc_name),
    username VARCHAR(255) REFERENCES users(username),
    event_description TEXT,
    key_words VARCHAR(255),
    event_category VARCHAR(255),
    process_name VARCHAR(255),
    process_id INTEGER
);

CREATE TABLE clients(
    id SERIAL PRIMARY KEY,
    pc_name VARCHAR(255) REFERENCES computers(pc_name),
    client_status VARCHAR(255) DEFAULT 'offline',
    client_version VARCHAR(255),
    last_message_time TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE security_levels(
    id SERIAL PRIMARY KEY,
    normalized_event_id INTEGER REFERENCES normalized_events(id),
    security_level VARCHAR(255) NOT NULL,
    security_value INTEGER CHECK (security_value BETWEEN 1 AND 5),
    security_level_description TEXT,
    auto_response VARCHAR(255),
    frontend_color VARCHAR(7) DEFAULT '#000000',
    created_at TIMESTAMPTZ DEFAULT NOW()
);