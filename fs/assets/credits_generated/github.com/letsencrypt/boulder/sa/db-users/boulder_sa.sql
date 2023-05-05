-- this file is run by test/create_db.sh to create users for each
-- component with the appropriate permissions.

-- These lines require MariaDB 10.1+
CREATE USER IF NOT EXISTS 'policy'@'localhost';
CREATE USER IF NOT EXISTS 'sa'@'localhost';
CREATE USER IF NOT EXISTS 'sa_ro'@'localhost';
CREATE USER IF NOT EXISTS 'ocsp_resp'@'localhost';
CREATE USER IF NOT EXISTS 'revoker'@'localhost';
CREATE USER IF NOT EXISTS 'importer'@'localhost';
CREATE USER IF NOT EXISTS 'mailer'@'localhost';
CREATE USER IF NOT EXISTS 'cert_checker'@'localhost';
CREATE USER IF NOT EXISTS 'ocsp_update'@'localhost';
CREATE USER IF NOT EXISTS 'ocsp_update_ro'@'localhost';
CREATE USER IF NOT EXISTS 'test_setup'@'localhost';
CREATE USER IF NOT EXISTS 'badkeyrevoker'@'localhost';

-- Storage Authority
GRANT SELECT,INSERT ON certificates TO 'sa'@'localhost';
GRANT SELECT,INSERT,UPDATE ON certificateStatus TO 'sa'@'localhost';
GRANT SELECT,INSERT ON issuedNames TO 'sa'@'localhost';
GRANT SELECT,INSERT,UPDATE ON certificatesPerName TO 'sa'@'localhost';
GRANT SELECT,INSERT,UPDATE ON registrations TO 'sa'@'localhost';
GRANT SELECT,INSERT on fqdnSets TO 'sa'@'localhost';
GRANT SELECT,INSERT,UPDATE ON orders TO 'sa'@'localhost';
GRANT SELECT,INSERT ON requestedNames TO 'sa'@'localhost';
GRANT SELECT,INSERT,DELETE ON orderFqdnSets TO 'sa'@'localhost';
GRANT SELECT,INSERT,UPDATE ON authz2 TO 'sa'@'localhost';
GRANT SELECT,INSERT ON orderToAuthz2 TO 'sa'@'localhost';
GRANT INSERT,SELECT ON serials TO 'sa'@'localhost';
GRANT SELECT,INSERT ON precertificates TO 'sa'@'localhost';
GRANT SELECT,INSERT ON keyHashToSerial TO 'sa'@'localhost';
GRANT SELECT,INSERT ON blockedKeys TO 'sa'@'localhost';
GRANT SELECT,INSERT,UPDATE ON newOrdersRL TO 'sa'@'localhost';
GRANT SELECT ON incidents TO 'sa'@'localhost';

GRANT SELECT ON certificates TO 'sa_ro'@'localhost';
GRANT SELECT ON certificateStatus TO 'sa_ro'@'localhost';
GRANT SELECT ON issuedNames TO 'sa_ro'@'localhost';
GRANT SELECT ON certificatesPerName TO 'sa_ro'@'localhost';
GRANT SELECT ON registrations TO 'sa_ro'@'localhost';
GRANT SELECT on fqdnSets TO 'sa_ro'@'localhost';
GRANT SELECT ON orders TO 'sa_ro'@'localhost';
GRANT SELECT ON requestedNames TO 'sa_ro'@'localhost';
GRANT SELECT ON orderFqdnSets TO 'sa_ro'@'localhost';
GRANT SELECT ON authz2 TO 'sa_ro'@'localhost';
GRANT SELECT ON orderToAuthz2 TO 'sa_ro'@'localhost';
GRANT SELECT ON serials TO 'sa_ro'@'localhost';
GRANT SELECT ON precertificates TO 'sa_ro'@'localhost';
GRANT SELECT ON keyHashToSerial TO 'sa_ro'@'localhost';
GRANT SELECT ON blockedKeys TO 'sa_ro'@'localhost';
GRANT SELECT ON newOrdersRL TO 'sa_ro'@'localhost';
GRANT SELECT ON incidents TO 'sa_ro'@'localhost';

-- OCSP Responder
GRANT SELECT ON certificateStatus TO 'ocsp_resp'@'localhost';

-- OCSP Generator Tool (Updater)
GRANT SELECT ON certificates TO 'ocsp_update'@'localhost';
GRANT SELECT,UPDATE ON certificateStatus TO 'ocsp_update'@'localhost';
GRANT SELECT ON precertificates TO 'ocsp_update'@'localhost';

GRANT SELECT ON certificates TO 'ocsp_update_ro'@'localhost';
GRANT SELECT ON certificateStatus TO 'ocsp_update_ro'@'localhost';
GRANT SELECT ON precertificates TO 'ocsp_update_ro'@'localhost';

-- Revoker Tool
GRANT SELECT ON registrations TO 'revoker'@'localhost';
GRANT SELECT ON certificates TO 'revoker'@'localhost';
GRANT SELECT ON precertificates TO 'revoker'@'localhost';
GRANT SELECT ON keyHashToSerial TO 'revoker'@'localhost';
GRANT SELECT,UPDATE ON blockedKeys TO 'revoker'@'localhost';

-- Expiration mailer
GRANT SELECT ON certificates TO 'mailer'@'localhost';
GRANT SELECT ON registrations TO 'mailer'@'localhost';
GRANT SELECT,UPDATE ON certificateStatus TO 'mailer'@'localhost';
GRANT SELECT ON fqdnSets TO 'mailer'@'localhost';

-- Cert checker
GRANT SELECT ON certificates TO 'cert_checker'@'localhost';
GRANT SELECT ON authz2 TO 'cert_checker'@'localhost';

-- Bad Key Revoker
GRANT SELECT,UPDATE ON blockedKeys TO 'badkeyrevoker'@'localhost';
GRANT SELECT ON keyHashToSerial TO 'badkeyrevoker'@'localhost';
GRANT SELECT ON certificateStatus TO 'badkeyrevoker'@'localhost';
GRANT SELECT ON precertificates TO 'badkeyrevoker'@'localhost';
GRANT SELECT ON registrations TO 'badkeyrevoker'@'localhost';

-- Test setup and teardown
GRANT ALL PRIVILEGES ON * to 'test_setup'@'localhost';
