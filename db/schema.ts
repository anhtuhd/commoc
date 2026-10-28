import { sqliteTable, text, integer } from 'drizzle-orm/sqlite-core';
export const shopState=sqliteTable('shop_state',{id:text('id').primaryKey(),body:text('body').notNull(),revision:integer('revision').notNull().default(0)});
