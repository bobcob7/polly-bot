CREATE TABLE IF NOT EXISTS agents (
	name STRING PRIMARY KEY,
	base_url STRING NOT NULL,
	query_formatter STRING DEFAULT '%s&q=%s'
);
CREATE TABLE IF NOT EXISTS subjects (
	name STRING PRIMARY KEY,
	search_text STRING NOT NULL,
	regex STRING,
	agent_name STRING NOT NULL REFERENCES agents (name)
);