-- +migrate Up
-- SQL in section 'Up' is executed when this migration is applied

CREATE TABLE `incidents` (
    `id` bigint(20) NOT NULL AUTO_INCREMENT,
    `serialTable` varchar(128) NOT NULL,
    `url` varchar(1024) NOT NULL,
    `renewBy` datetime NOT NULL,
    `enabled` boolean DEFAULT false,
    PRIMARY KEY (`id`)
) CHARSET=utf8mb4;

-- +migrate Down
-- SQL section 'Down' is executed when this migration is rolled back

DROP TABLE `incidents`;
