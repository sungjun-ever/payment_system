```aiignore
table users {
  id int [primary key]
  email varchar(100)
  name varchar(50)
  password varchar(255)
  created_at timestamp
  updaetd_at timestamp
  deleted_at timestamp
}

table products {
  id int [primary key]
  name varchar(100) [not null]
  description varchar(1000)
  price int64 [not null]
  status varchar(30) [not null, default:"ACTIVE"]
  created_at timestamp
  updated_at timestamp
  deleted_at timestamp
}

table orders {
  id int [primary key]
  order_no varchar(50) [not null, unique]
  user_id int [not null]
  status varchar(30) [not null, default: "PENDING"]
  total_amount int64 [not null]
  ordered_at timestamp [not null]
  created_at timestamp
  updated_at timestamp
  deleted_at timestamp
}

table order_items {
  id int [primary key]
  order_id int [not null]
  product_id int [not null]
  product_name varchar(100) [not null]
  unit_price int64
  quantity int
  total_price int64
}

table inventories {
  id int [primary key]
  product_id int [not null, unique]
  total_quantity int [not null]
  sold_quantity int [not null]
  reserved_quantity int [not null]
  created_at timestamp
  updated_at timestamp
  deleted_at timestamp
}

table inventory_jobs {
  id int64 [primary key]
  target varchar(50) [not null]
  operation varchar(100) [not null]
  status varchar(30) [not null]
  retry_count int [not null]
  next_retry_at timestamp
  last_error text
  payload text
  unique_key varchar(255) [not null, unique]
  created_at timestamp
  updated_at timestamp
}

table inventory_movements {
  id int [primary key]
  order_id int [not null, unique]
  product_id int [not null, unique]
  operation string [not null, unique]
  quantity int [not null]
  created_at timestamp
  updated_at timestamp
  deleted_at timestamp
}

table idempotency_keys {
  id int [primary key]
  user_id int [not null, unique]
  scope varchar(255) [not null, unique]
  key varchar(255) [not null, unique]
  request_hash char(64) [not null]
  status varchar(50) [not null]
  order_id int
  payment_id int
  response_code int
  response_body json
  created_at timestamp
  updated_at timestamp
  deleted_at timestamp
}

table payments {
  id int [primary key]
  user_id int
  payment_no varchar(50) [not null, unique]
  order_id int [not null, unique]
  status string [not null]
  paid_at timestamp
  cancelled_at timestamp
  refunded_at timestamp
}

table payment_attempts {
  id int [primary key]
  payment_id int [not null]
  client_idempotency_key varchar(255) [unique]
  action varchar(100) [not null]
  method varchar(100) [not null]
  status varchar(100) [not null]
  amount int
  provider varchar(50) [unique]
  provider_payment_id varchar(255) [unique]
  provider_idempotency_key varchar(255) [unique]
  failure_reason string
  created_at timestamp
  updated_at timestamp
  deleted_at timestamp
}

ref: users.id < orders.user_id
ref: users.id < payments.user_id
ref: users.id < idempotency_keys.user_id

ref: products.id - inventories.product_id
ref: products.id < inventory_movements.product_id
ref: products.id < order_items.product_id

ref: orders.id < order_items.order_id
ref: orders.id < inventory_movements.order_id
ref: orders.id < idempotency_keys.order_id
ref: orders.id < payments.order_id

ref: payments.id < payment_attempts.payment_id
```