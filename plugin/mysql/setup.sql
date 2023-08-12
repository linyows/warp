DROP DATABASE IF EXISTS `warp`;
CREATE DATABASE `warp`;

USE `warp`;

DROP TABLE IF EXISTS `connections`;
CREATE TABLE `connections` (
    `id` varchar(26) NOT NULL DEFAULT '',
    `mail_from` varchar(512) DEFAULT NULL,
    `mail_to` varchar(512) DEFAULT NULL,
    `occurred_at` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

DROP TABLE IF EXISTS `communications`;
CREATE TABLE `communications` (
    `id` varchar(26) NOT NULL DEFAULT '',
    `connection_id` varchar(26) NOT NULL DEFAULT '',
    `direction` varchar(2) NOT NULL DEFAULT '',
    `data` TEXT,
    `occurred_at` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

DROP USER IF EXISTS `warp`;
CREATE USER `warp`@`localhost` IDENTIFIED BY "PASSWORD";
GRANT ALL ON *.* TO `warp`@`localhost`;

FLUSH PRIVILEGES;
