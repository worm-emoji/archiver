package migrations

import "github.com/contextwtf/lanyard/migrate"

var Migrations = []migrate.Migration{
	{
		Name: "2022-10-30.0.init.sql",
		SQL: `
			CREATE TABLE bookmarks (
				url text NOT NULL UNIQUE,
				title text,
				description text,
				tags text[],
				ts timestamptz NOT NULL DEFAULT now()
			);
		`,
	},
}
