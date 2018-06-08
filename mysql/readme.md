# Usage Examples


# Create mySQL table

```sql
CREATE TABLE dbname.tablename (
		h int(10) NOT NULL AUTO_INCREMENT,
		x tinyint(4) NOT NULL DEFAULT '0',
		PRIMARY KEY (x),
		UNIQUE KEY h (h)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8;
```