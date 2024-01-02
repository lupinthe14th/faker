DROP TABLE IF EXISTS `panel_order_items`;
CREATE TABLE `panel_order_items` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `panel_order_id` INT NOT NULL,
  `question_id` INT NOT NULL,
  `order_index` INT NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  KEY `index_panel_order_items_on_panel_order_id` ( `panel_order_id` ),
  KEY `index_panel_order_items_on_question_id` ( `question_id` )
)ENGINE = InnoDB DEFAULT CHARSET=utf8mb4 COLLATE = utf8mb4_unicode_ci;
