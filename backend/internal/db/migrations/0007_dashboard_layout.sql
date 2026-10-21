-- Dashboard layout: column settings and widgets (openspec change dashboard-sections-editor).
-- "mobile_columns <= desktop_columns" is checked by the store, because ADD COLUMN
-- can't add a check across columns. layout_version only increases on a layout
-- save; it is the conflict check of the editor.

ALTER TABLE dashboards ADD COLUMN desktop_columns INTEGER NOT NULL DEFAULT 6 CHECK (desktop_columns BETWEEN 1 AND 12);
ALTER TABLE dashboards ADD COLUMN mobile_columns INTEGER NOT NULL DEFAULT 2 CHECK (mobile_columns BETWEEN 1 AND 12);
ALTER TABLE dashboards ADD COLUMN mobile_breakpoint INTEGER NOT NULL DEFAULT 768 CHECK (mobile_breakpoint IN (640, 768, 1024));
ALTER TABLE dashboards ADD COLUMN layout_version INTEGER NOT NULL DEFAULT 0;

-- A widget with parent_id NULL is at the top level. The composite foreign key
-- keeps every widget inside its parent's dashboard. config holds the settings
-- of the widget's kind as JSON.
CREATE TABLE dashboard_widgets (
	id           TEXT PRIMARY KEY,
	dashboard_id TEXT NOT NULL REFERENCES dashboards (id) ON DELETE CASCADE,
	parent_id    TEXT,
	kind         TEXT NOT NULL CHECK (kind IN ('section')),
	position     INTEGER NOT NULL,
	width        INTEGER NOT NULL CHECK (width BETWEEN 1 AND 12),
	config       TEXT NOT NULL DEFAULT '{}' CHECK (json_valid(config)),
	UNIQUE (id, dashboard_id),
	FOREIGN KEY (parent_id, dashboard_id) REFERENCES dashboard_widgets (id, dashboard_id) ON DELETE CASCADE
);

CREATE INDEX dashboard_widgets_parent ON dashboard_widgets (dashboard_id, parent_id, position);
