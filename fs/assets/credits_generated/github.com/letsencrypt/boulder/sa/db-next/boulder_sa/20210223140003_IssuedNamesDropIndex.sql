
-- +migrate Up
-- SQL in section 'Up' is executed when this migration is applied

ALTER TABLE issuedNames DROP INDEX `reversedName_renewal_notBefore_Idx`;

-- +migrate Down
-- SQL section 'Down' is executed when this migration is rolled back

ALTER TABLE issuedNames ADD INDEX `reversedName_renewal_notBefore_Idx` (`reversedName`,`renewal`,`notBefore`);
