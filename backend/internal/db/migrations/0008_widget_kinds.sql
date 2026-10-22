-- Widget kinds (openspec change widget-model).
-- Rebuilds dashboard_widgets without the check on kind, because SQLite can't
-- change a check in place. The store validates kinds, so new kinds don't need
-- another rebuild. Renaming the new table also renames its self-reference.

PRAGMA defer_foreign_keys = ON;

CREATE TABLE dashboard_widgets_new (
	id           TEXT PRIMARY KEY,
	dashboard_id TEXT NOT NULL REFERENCES dashboards (id) ON DELETE CASCADE,
	parent_id    TEXT,
	kind         TEXT NOT NULL,
	position     INTEGER NOT NULL,
	width        INTEGER NOT NULL CHECK (width BETWEEN 1 AND 12),
	config       TEXT NOT NULL DEFAULT '{}' CHECK (json_valid(config)),
	UNIQUE (id, dashboard_id),
	FOREIGN KEY (parent_id, dashboard_id) REFERENCES dashboard_widgets_new (id, dashboard_id) ON DELETE CASCADE
);

INSERT INTO dashboard_widgets_new (id, dashboard_id, parent_id, kind, position, width, config)
	SELECT id, dashboard_id, parent_id, kind, position, width, config FROM dashboard_widgets;

DROP TABLE dashboard_widgets;
ALTER TABLE dashboard_widgets_new RENAME TO dashboard_widgets;

CREATE INDEX dashboard_widgets_parent ON dashboard_widgets (dashboard_id, parent_id, position);
