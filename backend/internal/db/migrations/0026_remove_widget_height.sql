-- Content widgets no longer have a height (openspec change remove-widget-height).
-- A section with height "auto" whose only child is a content widget with a
-- fixed height takes that height over. Every other widget height is dropped.
-- A section stored before heights existed has no height, which means "auto".

UPDATE dashboard_widgets AS s
SET config = json_set(s.config, '$.height', (
	SELECT json_extract(c.config, '$.height') FROM dashboard_widgets AS c
	WHERE c.dashboard_id = s.dashboard_id AND c.parent_id = s.id
))
WHERE s.kind = 'section'
	AND COALESCE(json_extract(s.config, '$.height'), 'auto') = 'auto'
	AND (
		SELECT COUNT(*) FROM dashboard_widgets AS c
		WHERE c.dashboard_id = s.dashboard_id AND c.parent_id = s.id
	) = 1
	AND EXISTS (
		SELECT 1 FROM dashboard_widgets AS c
		WHERE c.dashboard_id = s.dashboard_id AND c.parent_id = s.id
			AND c.kind <> 'section' AND json_type(c.config, '$.height') = 'integer'
	);

UPDATE dashboard_widgets SET config = json_remove(config, '$.height') WHERE kind <> 'section';
