
-- +migrate Up
-- SQL in section 'Up' is executed when this migration is applied

ALTER TABLE authz2 DROP INDEX IF EXISTS token;
ALTER TABLE authz2 PARTITION BY RANGE(id) (
     PARTITION p_start VALUES LESS THAN MAXVALUE);

ALTER TABLE certificates DROP FOREIGN KEY IF EXISTS regId_certificates;
ALTER TABLE certificates DROP INDEX IF EXISTS serial, ADD INDEX serial (serial);
ALTER TABLE certificates PARTITION BY RANGE(id) (
    PARTITION p_start VALUES LESS THAN MAXVALUE);

ALTER TABLE fqdnSets DROP INDEX IF EXISTS serial, ADD INDEX serial (serial);
ALTER TABLE fqdnSets PARTITION BY RANGE(id) (
    PARTITION p_start VALUES LESS THAN MAXVALUE);

ALTER TABLE issuedNames PARTITION BY RANGE(id) (
    PARTITION p_start VALUES LESS THAN MAXVALUE);

ALTER TABLE orderFqdnSets DROP FOREIGN KEY IF EXISTS orderFqdnSets_orderID_orders;
ALTER TABLE orderFqdnSets DROP FOREIGN KEY IF EXISTS orderFqdnSets_registrationID_registrations;
ALTER TABLE orderFqdnSets PARTITION BY RANGE (id) (
    PARTITION p_start VALUES LESS THAN MAXVALUE);

ALTER TABLE orderToAuthz2 PARTITION BY RANGE COLUMNS(orderID, authzID) (
    PARTITION p_start VALUES LESS THAN (MAXVALUE, MAXVALUE));

-- Must be before orders, to remove the foreign key before partitioning orders.
ALTER TABLE requestedNames DROP FOREIGN KEY IF EXISTS orderID_orders;
ALTER TABLE requestedNames PARTITION BY RANGE (id) (
    PARTITION p_start VALUES LESS THAN MAXVALUE);

ALTER TABLE orders PARTITION BY RANGE (id) (
    PARTITION p_start VALUES LESS THAN MAXVALUE);

ALTER TABLE precertificates DROP FOREIGN KEY IF EXISTS regId_precertificates;
ALTER TABLE precertificates DROP INDEX IF EXISTS serial, ADD INDEX serial (serial);
ALTER TABLE precertificates PARTITION BY RANGE(id) (
    PARTITION p_start VALUES LESS THAN MAXVALUE);

-- +migrate Down
-- SQL section 'Down' is executed when this migration is rolled back

ALTER TABLE authz2 REMOVE PARTITIONING;
ALTER TABLE certificates REMOVE PARTITIONING;
ALTER TABLE fqdnSets REMOVE PARTITIONING;
ALTER TABLE issuedNames REMOVE PARTITIONING;
ALTER TABLE orderFqdnSets REMOVE PARTITIONING;
ALTER TABLE orderToAuthz2 REMOVE PARTITIONING;
ALTER TABLE orders REMOVE PARTITIONING;
ALTER TABLE precertificates REMOVE PARTITIONING;
ALTER TABLE requestedNames REMOVE PARTITIONING;
