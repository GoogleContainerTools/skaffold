-- +migrate Up
-- We use partitioning to clean up old data in these tables, and partitioning
-- is incompatible with unique indexes. Remove the unique indexes.

ALTER TABLE certificateStatus DROP INDEX IF EXISTS serial, ADD INDEX serial (serial);
ALTER TABLE certificateStatus PARTITION BY RANGE(id) (
    PARTITION p_start VALUES LESS THAN MAXVALUE);

ALTER TABLE certificates DROP INDEX IF EXISTS serial, ADD INDEX serial (serial);
ALTER TABLE certificates DROP FOREIGN KEY IF EXISTS regId_certificates;
ALTER TABLE certificates PARTITION BY RANGE(id) (
    PARTITION p_start VALUES LESS THAN MAXVALUE);

ALTER TABLE fqdnSets DROP INDEX IF EXISTS serial, ADD INDEX serial (serial);
ALTER TABLE fqdnSets PARTITION BY RANGE(id) (
    PARTITION p_start VALUES LESS THAN MAXVALUE);

ALTER TABLE precertificates DROP INDEX IF EXISTS serial, ADD INDEX serial (serial);
ALTER TABLE precertificates DROP FOREIGN KEY IF EXISTS regId_precertificates;
ALTER TABLE precertificates PARTITION BY RANGE(id) (
    PARTITION p_start VALUES LESS THAN MAXVALUE);

-- +migrate Down
-- SQL section 'Down' is executed when this migration is rolled back

