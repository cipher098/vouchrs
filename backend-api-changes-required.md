# Backend API Changes Required for Frontend

**Date:** 2026-03-28
**Context:** The frontend (Next.js) has been built against the Swagger doc (`doc.json`). Most endpoints are sufficient. Below are the specific changes/additions needed.

---

## Summary

| # | Change | Type | Priority | Effort |
|---|--------|------|----------|--------|
| 1 | Add `GET /api/v1/brands` endpoint | **New endpoint** | P0 — blocks frontend | Small |
| 2 | Add `GET /api/v1/listings/recommended-price` endpoint | **New endpoint** | P1 — blocks seller flow | Small |
| 3 | Include `status` + `lockExpiresAt` in marketplace listings | **Response change** | P1 — blocks lock UI | Trivial |
| 4 | Add `brand_name` and `brand_color` to marketplace response | **Response change** | P1 — blocks browse UI | Small |
| 5 | Add PhonePe return URL param to initiate-buy | **Request change** | P1 — blocks payment flow | Trivial |
| 6 | Rename "CardSwap" → "Vouchrs" in API descriptions/responses | **Cosmetic** | P2 | Trivial |

---

## Change 1: Add `GET /api/v1/brands` (P0)

**Why:** The frontend renders a brand grid on the landing page and filter chips on the browse page. Brands are referenced by UUID throughout the API, but there's no way to fetch the brand catalog (name, color, active status).

**Spec:**

```
GET /api/v1/brands

Auth: None required
```

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "Amazon",
      "color": "#ff9900",
      "is_active": true,
      "listing_count": 19
    },
    {
      "id": "660e8400-e29b-41d4-a716-446655440001",
      "name": "Flipkart",
      "color": "#2874f0",
      "is_active": true,
      "listing_count": 8
    }
  ]
}
```

**Fields:**
| Field | Type | Description |
|-------|------|-------------|
| `id` | UUID | Brand identifier |
| `name` | string | Display name |
| `color` | string | Hex color for UI gradient/icon |
| `is_active` | boolean | Whether brand is currently accepting listings |
| `listing_count` | integer | Number of LIVE listings for this brand (optional but helpful for display) |

---

## Change 2: Add `GET /api/v1/listings/recommended-price` (P1)

**Why:** The PRD specifies that sellers see a recommended price before choosing to accept (pool) or set a custom price. The frontend needs to show this before the seller submits `POST /api/v1/listings`.

**Spec:**

```
GET /api/v1/listings/recommended-price?brand_id={uuid}&face_value={number}

Auth: Bearer token required
```

**Query params:**
| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `brand_id` | UUID | Yes | Brand UUID |
| `face_value` | number | Yes | Card face value in INR |

**Response:**
```json
{
  "success": true,
  "data": {
    "recommended_discount_pct": 15.0,
    "seller_price": 850.00,
    "seller_payout": 845.75,
    "buyer_price": 854.25,
    "platform_fee_per_side": 4.25,
    "avg_sell_time_mins": 45
  }
}
```

**Fields:**
| Field | Type | Description |
|-------|------|-------------|
| `recommended_discount_pct` | number | Recommended discount % off face value |
| `seller_price` | number | What the seller lists at (face_value - discount) |
| `seller_payout` | number | What seller receives after 0.5% fee |
| `buyer_price` | number | What buyer pays including 0.5% fee |
| `platform_fee_per_side` | number | 0.5% fee amount |
| `avg_sell_time_mins` | number | Average time to sell at this price (for display) |

**Notes:** In V1 this can return a fixed discount (the platform standard rate). Dynamic pricing is a future feature.

---

## Change 3: Include `status` + `lockExpiresAt` in marketplace listings (P1)

**Why:** The browse page shows a "Reserving..." badge on locked cards with a greyed-out state. The frontend needs to know if a listing is LOCKED and when the lock expires to display this correctly.

**Current:** The `Listing` entity has `status` and `lockExpiresAt` fields, but it's unclear if these are included in `GET /api/v1/marketplace` response for individual listings.

**Required:** Ensure the `individualListings[]` array in the marketplace response includes:
- `status` — at minimum "LIVE" or "LOCKED" (exclude SOLD/CANCELLED/EXPIRED from marketplace results)
- `lockExpiresAt` — ISO timestamp, null if not locked

```json
{
  "individualListings": [
    {
      "id": "...",
      "brandID": "...",
      "faceValue": 1000,
      "buyerPrice": 854.25,
      "sellerPayout": 845.75,
      "isPool": false,
      "status": "LOCKED",
      "lockExpiresAt": "2026-03-28T15:30:00Z"
    }
  ]
}
```

---

## Change 4: Add brand metadata to marketplace response (P1)

**Why:** The browse page groups cards by brand and shows brand name + color + logo. Without brand metadata in the marketplace response, the frontend has to make a separate `GET /api/v1/brands` call and join client-side.

**Option A (preferred):** If Change 1 (brands endpoint) is implemented, frontend will join client-side. No change needed here.

**Option B:** Embed brand info in pool groups and listings:

```json
{
  "poolGroups": [
    {
      "id": "...",
      "brandID": "...",
      "brand_name": "Amazon",
      "brand_color": "#ff9900",
      "faceValue": 1000,
      "buyerPrice": 854.25,
      "activeCount": 14
    }
  ]
}
```

**Recommendation:** Implement Change 1, frontend will handle the join.

---

## Change 5: Add return URL to initiate-buy (P1)

**Why:** After PhonePe payment completes, the buyer is redirected back to our website. The frontend needs to tell the backend what URL to use as the PhonePe return URL so the buyer lands on the correct purchase confirmation page.

**Current `POST /api/v1/listings/{id}/buy`** — no return URL parameter.

**Add to request:**
```json
{
  "return_url": "https://vouchrs.in/purchase/confirm?txn_id=123"
}
```

Or the backend can construct this from a configured frontend base URL + transaction_id. Either way, the PhonePe redirect config needs a `merchantRedirectUrl` that points back to the frontend.

**If backend constructs it:** Add `return_url` to the response so frontend knows where to redirect on manual fallback:
```json
{
  "transaction_id": "...",
  "amount": 854.25,
  "lock_expires_at": "...",
  "payment_url": "https://api.phonepe.com/...",
  "return_url": "https://vouchrs.in/purchase/confirm?txn_id=..."
}
```

---

## Change 6: Rename "CardSwap" → "Vouchrs" (P2)

**Current:** The Swagger doc title says "CardSwap India API", support email is "support@cardswap.in", and descriptions reference "CardSwap pool", "CardSwap Support" etc.

**Change:** Update all references to "Vouchrs":
- API title: "Vouchrs API"
- Contact email: "support@vouchrs.in"
- Description references: "Vouchrs pool", "Vouchrs Support"
- `isPool` description: "true = Vouchrs pool" (currently "true = CardSwap pool")

This is cosmetic but important if any of these strings appear in user-facing API responses or error messages.

---

## What's Already Working (No Changes Needed)

These endpoints are fully sufficient as-is:

| Endpoint | Frontend Use |
|----------|-------------|
| `POST /api/v1/auth/request-otp` | Login modal — send OTP |
| `POST /api/v1/auth/verify-otp` | Login modal — verify & get JWT |
| `POST /api/v1/auth/refresh` | Token refresh in background |
| `POST /api/v1/auth/logout` | Logout button |
| `GET /api/v1/users/me` | Show logged-in user info |
| `GET /api/v1/marketplace` | Browse page (with changes 3+4) |
| `GET /api/v1/listings/{id}` | Single listing detail |
| `POST /api/v1/listings/{id}/buy` | Purchase flow — initiate (with change 5) |
| `GET /api/v1/transactions/{id}` | Poll payment status after PhonePe return |
| `POST /api/v1/transactions/{id}/confirm` | Buyer confirms redemption |
| `POST /api/v1/webhooks/phonepe` | Server-side webhook (no frontend change) |
| `POST /api/v1/listings` | Seller creates listing |
| `DELETE /api/v1/listings/{id}` | Seller cancels listing |
| `GET /api/v1/dashboard/listings` | Dashboard — my listings tab |
| `GET /api/v1/dashboard/purchases` | Dashboard — my purchases tab |
| `GET /api/v1/dashboard/requests` | Dashboard — my requests tab |
| `POST /api/v1/buy-requests` | "Get Notified" button on browse page |
| `GET /api/v1/buy-requests` | Dashboard — my buy requests |
| `DELETE /api/v1/buy-requests/{id}` | Cancel a buy request |
| `POST /api/v1/card-requests` | "Request Card" button on browse page |

---

## Frontend Integration Timeline

Once the backend changes are ready:

1. **Phase 1** (can start now): Wire auth flow (OTP login), marketplace browse with real data, purchase flow with PhonePe redirect
2. **Phase 2** (needs Change 1+2): Seller listing flow with recommended pricing, brand-aware UI
3. **Phase 3** (needs dashboard): User dashboard with all 3 tabs
