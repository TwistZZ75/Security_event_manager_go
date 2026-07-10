CREATE TYPE user_role AS ENUM ('admin', 'analyst', 'viewer');
CREATE TABLE users(
    id SERIAL NOT NULL,
    username varchar(50) NOT NULL,
    email varchar(255) NOT NULL,
    password_hash varchar(255) NOT NULL,
    role user_role NOT NULL DEFAULT 'viewer'::user_role,
    is_active boolean NOT NULL DEFAULT true,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    last_login_at timestamp with time zone,
    PRIMARY KEY(id),
    CONSTRAINT users_username_length CHECK ((length((username)::text) >= 3) AND (length((username)::text) <= 50))
);
CREATE UNIQUE INDEX users_username_unique ON public.users USING btree (username);
CREATE UNIQUE INDEX users_email_unique ON public.users USING btree (email);
CREATE INDEX idx_users_username ON public.users USING btree (username);
CREATE INDEX idx_users_email ON public.users USING btree (email);
CREATE INDEX idx_users_is_active ON public.users USING btree (is_active);

CREATE TABLE rules(
    id varchar(255) NOT NULL,
    name varchar(500) NOT NULL,
    enabled boolean DEFAULT false,
    severity varchar(50) NOT NULL,
    rule_definition jsonb NOT NULL,
    tags text[],
    created_by varchar(255),
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_by varchar(255),
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    last_triggered timestamp without time zone,
    trigger_count bigint DEFAULT 0,
    PRIMARY KEY(id),
    CONSTRAINT rules_severity_check CHECK ((severity)::text = ANY ((ARRAY['low'::character varying, 'medium'::character varying, 'high'::character varying, 'critical'::character varying])::text[]))
);
CREATE INDEX idx_rules_created_at ON public.rules USING btree (created_at DESC);

CREATE TABLE alerts(
    id SERIAL NOT NULL,
    rule_id varchar(255) NOT NULL,
    rule_name varchar(500),
    severity varchar(50) NOT NULL,
    title varchar(500) NOT NULL,
    description text,
    event_data jsonb,
    status varchar(50) DEFAULT 'open'::character varying,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    acknowledged_at timestamp without time zone,
    acknowledged_by varchar(255),
    resolved_at timestamp without time zone,
    resolved_by varchar(255),
    notes text,
    PRIMARY KEY(id),
    CONSTRAINT alerts_rule_id_fkey FOREIGN key(rule_id) REFERENCES rules(id),
    CONSTRAINT alerts_severity_check CHECK ((severity)::text = ANY ((ARRAY['low'::character varying, 'medium'::character varying, 'high'::character varying, 'critical'::character varying])::text[])),
    CONSTRAINT alerts_status_check CHECK ((status)::text = ANY ((ARRAY['open'::character varying, 'acknowledged'::character varying, 'resolved'::character varying, 'false_positive'::character varying])::text[]))
);
CREATE INDEX idx_alerts_status_created ON public.alerts USING btree (status, created_at DESC);

CREATE TABLE actions_log(
    id SERIAL NOT NULL,
    alert_id integer NOT NULL,
    action_type varchar(100) NOT NULL,
    target varchar(500),
    parameters jsonb,
    status varchar(50) NOT NULL DEFAULT 'pending'::character varying,
    executed_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    result text,
    error text,
    PRIMARY KEY(id),
    CONSTRAINT actions_log_alert_id_fkey FOREIGN key(alert_id) REFERENCES alerts(id),
    CONSTRAINT actions_log_status_check CHECK ((status)::text = ANY ((ARRAY['pending'::character varying, 'success'::character varying, 'failed'::character varying])::text[]))
);
CREATE INDEX idx_actions_log_executed_at ON public.actions_log USING btree (executed_at DESC);

CREATE TABLE agent_commands(
    id SERIAL NOT NULL,
    hostname varchar(255) NOT NULL,
    command_type varchar(100) NOT NULL,
    parameters jsonb,
    alert_id integer,
    priority varchar(50) DEFAULT 'medium'::character varying,
    status varchar(50) DEFAULT 'pending'::character varying,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    sent_at timestamp without time zone,
    completed_at timestamp without time zone,
    result text,
    error text,
    PRIMARY KEY(id),
    CONSTRAINT agent_commands_alert_id_fkey FOREIGN key(alert_id) REFERENCES alerts(id),
    CONSTRAINT agent_commands_priority_check CHECK ((priority)::text = ANY ((ARRAY['low'::character varying, 'medium'::character varying, 'high'::character varying, 'critical'::character varying])::text[])),
    CONSTRAINT agent_commands_status_check CHECK ((status)::text = ANY ((ARRAY['pending'::character varying, 'sent'::character varying, 'in_progress'::character varying, 'success'::character varying, 'failed'::character varying, 'cancelled'::character varying])::text[]))
);
CREATE INDEX idx_agent_commands_hostname ON public.agent_commands USING btree (hostname);
CREATE INDEX idx_agent_commands_status ON public.agent_commands USING btree (status);
CREATE INDEX idx_agent_commands_hostname_status ON public.agent_commands USING btree (hostname, status);
CREATE INDEX idx_agent_commands_created_at ON public.agent_commands USING btree (created_at DESC);
CREATE INDEX idx_agent_commands_alert_id ON public.agent_commands USING btree (alert_id);
COMMENT ON TABLE agent_commands IS 'Commands queue for agents to execute';
COMMENT ON COLUMN agent_commands.priority IS 'Command priority: critical commands execute first';
COMMENT ON COLUMN agent_commands.status IS 'Command lifecycle: pending -> sent -> in_progress -> success/failed';

CREATE TABLE agents(
    id SERIAL NOT NULL,
    agent_id varchar(255) NOT NULL,
    hostname varchar(255) NOT NULL,
    os varchar(50) NOT NULL,
    os_version varchar(100),
    agent_version varchar(50),
    ip_address varchar(45),
    metadata jsonb,
    status varchar(50) DEFAULT 'offline'::character varying,
    registered_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    last_seen timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY(id),
    CONSTRAINT agents_status_check CHECK ((status)::text = ANY ((ARRAY['online'::character varying, 'offline'::character varying, 'error'::character varying])::text[]))
);
CREATE UNIQUE INDEX agents_agent_id_key ON public.agents USING btree (agent_id);
CREATE UNIQUE INDEX agents_hostname_key ON public.agents USING btree (hostname);
CREATE INDEX idx_agents_hostname ON public.agents USING btree (hostname);
CREATE INDEX idx_agents_status ON public.agents USING btree (status);
CREATE INDEX idx_agents_last_seen ON public.agents USING btree (last_seen DESC);
COMMENT ON TABLE agents IS 'Registered SIEM agents on client machines';
COMMENT ON COLUMN agents.last_seen IS 'Last time agent communicated with server';



CREATE TABLE normalized_events(
    id varchar(64) NOT NULL,
    pc_name varchar(255),
    username varchar(255),
    event_description text,
    event_category varchar(255),
    process_name varchar(255),
    severity text DEFAULT 'INFO'::text,
    timestamp timestamp with time zone NOT NULL,
    os varchar(255) NOT NULL,
    "source" varchar(255),
    raw_log text,
    PRIMARY KEY(id)
);
COMMENT ON COLUMN normalized_events."source" IS 'источник лога';

CREATE TABLE refresh_tokens(
    id SERIAL NOT NULL,
    user_id bigint NOT NULL,
    token varchar(64) NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    PRIMARY KEY(id),
    CONSTRAINT refresh_tokens_user_id_fkey FOREIGN key(user_id) REFERENCES users(id)
);
CREATE UNIQUE INDEX refresh_tokens_user_unique ON public.refresh_tokens USING btree (user_id);
CREATE UNIQUE INDEX refresh_tokens_token_unique ON public.refresh_tokens USING btree (token);
CREATE INDEX idx_refresh_tokens_token ON public.refresh_tokens USING btree (token);
CREATE INDEX idx_refresh_tokens_expires_at ON public.refresh_tokens USING btree (expires_at);

CREATE TABLE rule_state(
    id SERIAL NOT NULL,
    rule_id varchar(255) NOT NULL,
    group_key varchar(500) NOT NULL,
    counter integer DEFAULT 0,
    first_seen timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    last_seen timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    state_data jsonb,
    expires_at timestamp without time zone NOT NULL,
    PRIMARY KEY(id),
    CONSTRAINT rule_state_rule_id_fkey FOREIGN key(rule_id) REFERENCES rules(id)
);
CREATE UNIQUE INDEX rule_state_rule_id_group_key_key ON public.rule_state USING btree (rule_id, group_key);
CREATE INDEX idx_rule_state_rule_group ON public.rule_state USING btree (rule_id, group_key);



