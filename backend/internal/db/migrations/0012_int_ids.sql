-- Integer IDs for folders, links and tags (openspec change linkwarden-api), for
-- apps whose clients expect numbers. Existing rows are numbered in creation
-- order. id_counters holds the last number given out; it only goes up, so a
-- number is never reused, also after the row with the highest one is deleted.
-- The triggers number every new row, whichever code inserts it.

ALTER TABLE favourite_nodes ADD COLUMN num INTEGER;
ALTER TABLE tags ADD COLUMN num INTEGER;

UPDATE favourite_nodes SET num = o.n
FROM (SELECT id, ROW_NUMBER() OVER (ORDER BY created_at, id) AS n FROM favourite_nodes) AS o
WHERE o.id = favourite_nodes.id;

UPDATE tags SET num = o.n
FROM (SELECT id, ROW_NUMBER() OVER (ORDER BY created_at, id) AS n FROM tags) AS o
WHERE o.id = tags.id;

CREATE UNIQUE INDEX favourite_nodes_num ON favourite_nodes (num);
CREATE UNIQUE INDEX tags_num ON tags (num);

CREATE TABLE id_counters (
	name TEXT PRIMARY KEY,
	last INTEGER NOT NULL
);

INSERT INTO id_counters (name, last) VALUES
	('nodes', (SELECT COALESCE(MAX(num), 0) FROM favourite_nodes)),
	('tags', (SELECT COALESCE(MAX(num), 0) FROM tags));

CREATE TRIGGER favourite_nodes_assign_num AFTER INSERT ON favourite_nodes
WHEN NEW.num IS NULL
BEGIN
	UPDATE id_counters SET last = last + 1 WHERE name = 'nodes';
	UPDATE favourite_nodes SET num = (SELECT last FROM id_counters WHERE name = 'nodes') WHERE id = NEW.id;
END;

CREATE TRIGGER tags_assign_num AFTER INSERT ON tags
WHEN NEW.num IS NULL
BEGIN
	UPDATE id_counters SET last = last + 1 WHERE name = 'tags';
	UPDATE tags SET num = (SELECT last FROM id_counters WHERE name = 'tags') WHERE id = NEW.id;
END;
