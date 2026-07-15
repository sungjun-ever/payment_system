## 공통 변수
```
BASE_URL=http://localhost:8080
ACCESS_TOKEN=
IDEMPOTENCY_KEY=
```
### 인증/회원
#### 회원가입
```
curl -X POST "{BASE_URL}/api/v1/register" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "name": "홍길동",
    "password": "password1234",
    "password_confirm": "password1234"
  }'
```

#### 로그인
```
curl -X POST "{BASE_URL}/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password1234"
  }'
```

#### 토큰 재발급
```
curl -X POST "{BASE_URL}/api/v1/auth/refresh" \
  -b refresh_token={refresh_token} \
  -c refresh_token={refresh_token}
```

#### 로그아웃
```
curl -X DELETE "{BASE_URL}/api/v1/auth/logout" \
  -H "Authorization: Bearer {ACCESS_TOKEN}" \
```

### 멱등키
#### 멱등키 생성
`origin/action` 예시
- 주문 요청: `order/create`
- 주문 취소: `order/cancel`
- 결제 요청: `payment/create`
- 환불 요청: `payment/refund`
```
curl -X POST "{BASE_URL}/api/v1/idempotencies" \
  -H "Authorization: Bearer {ACCESS_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "origin": "order",
    "action": "create"
  }'
```

### 상품
#### 상품 생성
```
curl -X POST "{BASE_URL}/api/v1/products" \
  -H "Authorization: Bearer {ACCESS_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "테스트 상품",
    "description": "상품 설명",
    "price": 10000,
    "status": "ACTIVE",
    "inventory": {
      "total_quantity": 100,
      "reserved_quantity": 0,
      "sold_quantity": 0
    }
  }'
```

#### 상품 조회
```
curl -X GET "$BASE_URL/api/v1/products/1" \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

#### 상품 수정
```
curl -X PUT "{BASE_URL}/api/v1/products/1" \
  -H "Authorization: Bearer {ACCESS_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "수정된 상품",
    "description": "수정된 상품 설명",
    "price": 12000,
    "status": "ACTIVE",
    "inventory": {
      "total_quantity": 150
    }
  }'
```

#### 상품 삭제
```
curl -X DELETE "{BASE_URL}/api/v1/products/1" \
  -H "Authorization: Bearer {ACCESS_TOKEN}"
```

### 주문
#### 주문 요청
```
curl -X POST "{BASE_URL}/api/v1/orders" \
  -H "Authorization: Bearer {ACCESS_TOKEN}" \
  -H "Idempotency-Key: {IDEMPOTENCY_KEY}" \
  -H "Content-Type: application/json" \
  -d '{
    "order_no": "ORDER-20260713-0001",
    "total_amount": 20000,
    "ordered_at": "2026-07-13 12:00:00",
    "ordered_items": [
      {
        "product_id": 1,
        "product_name": "테스트 상품",
        "unit_price": 10000,
        "quantity": 2,
        "total_price": 20000
      }
    ]
  }'
```

#### 주문 취소
```
curl -X DELETE "{BASE_URL}/api/v1/orders/1?orderNo=ORDER-20260713-0001" \
  -H "Authorization: Bearer {ACCESS_TOKEN}" \
  -H "Idempotency-Key: {IDEMPOTENCY_KEY}"
```

### 결제
#### 결제 요청
```
curl -X POST "{BASE_URL}/api/v1/payments" \
  -H "Authorization: Bearer {ACCESS_TOKEN}" \
  -H "Idempotency-Key: {IDEMPOTENCY_KEY}" \
  -H "Content-Type: application/json" \
  -d '{
    "payment_no": "PAY-20260713-0001",
    "order_id": 1,
    "method": "CARD",
    "action": "PAY",
    "amount": 20000,
    "provider": "TOSS"
  }'
```

#### 결제 환불
```
curl -X PUT "{BASE_URL}/api/v1/payments/1/refund?order_no=ORDER-20260713-0001" \
  -H "Authorization: Bearer {ACCESS_TOKEN}" \
  -H "Idempotency-Key: {IDEMPOTENCY_KEY}" 
```