-- 000005_mall_orders 回滚：按依赖倒序删除
DROP TABLE IF EXISTS point_batch;
DROP TABLE IF EXISTS redeem_order;
DROP TABLE IF EXISTS product_sku;
DROP TABLE IF EXISTS product;
DROP TABLE IF EXISTS merchant;
