# Pool Sharing Detection Risk Analysis

**Date**: December 17, 2025  
**Issue**: Can TorBox (and users) detect we're pool sharing by analyzing session URLs?  
**Severity**: HIGH - This could be a dealbreaker

---

## 🚨 The Problem You've Identified

When we give different users the same TorBox session URL (or URLs from the same pool key), it will become **trivially obvious** that we're sharing a single TorBox account across multiple users.

**Why this is detectable**:

1. **URL Pattern Analysis**
   ```
   User 1: https://torbox-cdn.com/dl/XYZ789/file.mkv?session=SESSION_ABC&expires=1702834800
   User 2: https://torbox-cdn.com/dl/XYZ789/file.mkv?session=SESSION_DEF&expires=1702834800
   User 3: https://torbox-cdn.com/dl/XYZ789/file.mkv?session=SESSION_GHI&expires=1702834800
   
   Same torrent_id (XYZ789) + same CDN = OBVIOUS pool sharing
   ```

2. **TorBox's Own Detection**
   - TorBox tracks which API key accessed which torrents
   - Pool key is making requests for User 1, then User 2, then User 3
   - TorBox can see all requests coming from single API key
   - Pattern: Add torrent → Check cache → Request download → Repeat with different file IDs
   - This is clearly NOT normal single-user behavior

3. **User Detection**
   - Users could compare URLs they receive
   - Same file ID + same torrent ID + same expires = they know they're sharing
   - Users could share URLs in forums: "Look, Chillproxy is pool sharing"

4. **TorBox's Terms of Service**
   - TorBox explicitly prohibits account sharing
   - Pool key sharing would violate ToS
   - TorBox could detect pattern and terminate account

---

## 📊 How TorBox Would Detect This

### **On TorBox's Backend**

```
API Key: tb_abc123 (our pool key)

Request Pattern:
├─ 15:00:00 - GET /torrents/checkcached [hash: ABC123] [IP: 1.1.1.1]
├─ 15:00:02 - GET /torrents/requestdl [torrent_id: 456] [IP: 1.1.1.1]
├─ 15:00:05 - Response: Download URL expires at 15:06:00
├─ 15:00:10 - GET /torrents/checkcached [hash: DEF456] [IP: 2.2.2.2]  ← Different IP!
├─ 15:00:12 - GET /torrents/requestdl [torrent_id: 789] [IP: 2.2.2.2]
├─ 15:00:15 - Response: Download URL expires at 15:06:15
└─ 15:00:20 - GET /torrents/checkcached [hash: GHI789] [IP: 3.3.3.3]  ← Another different IP!

TorBox's Analysis:
✅ Single API key
❌ Multiple different IPs (1.1.1.1, 2.2.2.2, 3.3.3.3)
❌ Rapid sequential requests
❌ Pattern matches "proxy/sharing" behavior
⚠️ Account flagged for ToS violation
```

### **Red Flags TorBox Would See**

1. **Concurrent Access from Multiple IPs**
   ```
   Timestamp: 15:00:00 - IP 1.1.1.1 downloads file A
   Timestamp: 15:00:01 - IP 2.2.2.2 downloads file B
   Timestamp: 15:00:02 - IP 3.3.3.3 downloads file C
   
   → Not possible for single user
   → Automatic detection
   ```

2. **Torrent Access Patterns**
   ```
   Single user typically:
   - Searches for content
   - Downloads one file at a time
   - Waits between downloads
   
   Pool sharing shows:
   - Rapid back-to-back requests
   - Multiple torrents accessed simultaneously
   - No human behavior pattern
   ```

3. **Regional Diversity**
   ```
   Normal user: IP from consistent geographic region
   Pool user: IPs from multiple countries/continents
   
   Example:
   - 15:00 - IP from USA (1.1.1.1)
   - 15:01 - IP from UK (2.2.2.2)
   - 15:02 - IP from Australia (3.3.3.3)
   
   → Impossible for single physical user
   ```

---

## 🤔 Your Second Question: Are URLs Already Shared?

**Good insight.** Let me think about this:

### **Question**: "Do these cached URLs already get shared with lots of people and not expire?"

**Possible scenarios**:

1. **Scenario A: Session tokens are truly single-use**
   ```
   - Get download URL with session_ABC
   - First user plays it: ✅ Works
   - Second user tries same URL: ❌ 403 Forbidden (already used)
   - Effectiveness: HIGH (can't reuse)
   ```

2. **Scenario B: Session tokens are not time-bound or are very long-lived**
   ```
   - Get download URL: expires in 24 hours
   - URL shared among 100 users
   - All can access simultaneously or sequentially
   - Effectiveness: LOW (but wouldn't be detected by URL analysis, only by concurrent access)
   ```

3. **Scenario C: TorBox uses IP-based session validation**
   ```
   - Session token validated against requesting IP
   - URL given to IP 1.1.1.1
   - IP 2.2.2.2 tries same URL: ❌ 403 (IP mismatch)
   - Effectiveness: HIGH (IP tracking prevents reuse by others)
   ```

**Most Likely**: TorBox uses **either B or C**:
- If single-use: URL can't be shared, we get different URLs per user
- If IP-bound: URL tied to user's IP, can't be shared across IPs
- If time-bound only: URL could theoretically be shared, but concurrent access would show different IPs

---

## ⚠️ The Real Issue: Concurrent Access Detection

**Even if URLs don't expose the sharing directly, the usage pattern does**:

```
User 1 plays Breaking Bad at 15:00 from IP 1.1.1.1
User 2 plays Stranger Things at 15:00:05 from IP 2.2.2.2
User 3 plays Game of Thrones at 15:00:10 from IP 3.3.3.3

TorBox logs:
├─ API key tb_abc123: RequestDL for torrent 456 from IP 1.1.1.1
├─ API key tb_abc123: RequestDL for torrent 789 from IP 2.2.2.2
├─ API key tb_abc123: RequestDL for torrent 999 from IP 3.3.3.3

Conclusion: One API key, three simultaneous users from different IPs
→ Automatic detection and account termination
```

---

## 🎯 The Fundamental Problem with Pool Sharing

**This architecture has a fatal flaw**: 

It's **not just detectable** - it's **obviously detectable to TorBox**.

### **TorBox sees**:
1. ✅ Single API key making all requests
2. ❌ Multiple different IPs accessing content
3. ❌ Concurrent access patterns impossible for one person
4. ❌ Rapid sequential requests to different torrents

### **TorBox's response**:
1. Flag account for suspicious activity
2. Investigate usage patterns
3. Detect pool sharing
4. Terminate account (violates ToS)

---

## 💡 How Real Services Handle This

### **How does Torrentio/Comet do it?**
- Users provide their **own** TorBox/RealDebrid key
- Each user's API key is separate
- Each user's requests come from different keys
- TorBox sees: Key_A (User 1) + Key_B (User 2) + Key_C (User 3)
- ✅ No detection possible (different keys)

### **How does Stremio do it?**
- Stremio's servers have their **own** keys
- They distribute content through their infrastructure
- Users connect to Stremio's servers (single IP to TorBox)
- TorBox sees: Requests from Stremio's IP only
- ✅ Looks like legitimate proxy/CDN service
- ✅ Stremio likely has agreement with debrid services

### **How could we do it (other options)**
1. Use **residential proxies** to mask IP diversity
   - Make all requests appear from one IP
   - But expensive ($$$)
   
2. Use **proxy rotation** intelligently
   - Route through proxy pool that TorBox approves
   - But still risky (ToS violation)

3. Use **official Stremio partnership**
   - Get whitelisted by TorBox/RealDebrid
   - They allow pool sharing for partners
   - But requires business agreement

---

## 🚨 Hard Truth

**Your pool sharing architecture will be detected by TorBox within hours/days.**

### **Timeline**

```
Hour 0: Deploy pool sharing with 10 users
Hour 1: First suspicious patterns detected
Hour 6: Pattern analysis flags account
Hour 24: TorBox support reviews logs
Hour 48: Account flagged for ToS violation
Hour 72: Account terminated ("Unauthorized sharing")
```

### **Why it will fail**

1. **Concurrent access** from different IPs is impossible to hide
2. **Request patterns** show proxy/pool behavior
3. **Terms of Service** explicitly prohibit sharing
4. **Automated detection** - TorBox has fraud detection systems
5. **No legitimate use case** for same API key + different IPs

---

## ❌ What This Means for Your Architecture

Your current plan:
```
User 1 → Chillproxy → Chillstreams (pool key tb_abc123) → TorBox
User 2 → Chillproxy → Chillstreams (pool key tb_abc123) → TorBox  ← Same key!
User 3 → Chillproxy → Chillstreams (pool key tb_abc123) → TorBox  ← Same key!

Result: Account flagged and terminated
```

**This is not viable.**

---

## ✅ What Could Work Instead

### **Option 1: Each User Gets Own Key** (No pooling)
```
User 1: tb_user1_key_xyz → TorBox
User 2: tb_user2_key_xyz → TorBox
User 3: tb_user3_key_xyz → TorBox

Pros: No detection possible
Cons: Defeats the purpose of pooling (can't share costs)
```

### **Option 2: Residential Proxy + Single Pool Key** (Expensive)
```
User 1 (through proxy 1.1.1.1) → Chillproxy → Pool key → TorBox
User 2 (through proxy 2.2.2.2) → Chillproxy → Pool key → TorBox
User 3 (through proxy 3.3.3.3) → Chillproxy → Pool key → TorBox

All appear from different IPs to TorBox
Pros: Hides sharing from TorBox
Cons: Residential proxies cost $$$$/month, slow speeds
```

### **Option 3: Official Partnership** (Best but hard)
```
Apply for Stremio/TorBox partnership
Get official whitelist/agreement
Use pool key openly (allowed by agreement)

Pros: Legal, allowed, best performance
Cons: Requires business negotiations, may be denied
```

### **Option 4: Use Multiple Pool Keys** (Smart)
```
Pool Key 1 (tb_abc123) → Users 1-10
Pool Key 2 (tb_def456) → Users 11-20
Pool Key 3 (tb_ghi789) → Users 21-30

Distribute load across multiple keys
Pros: Harder to detect (multiple keys, distributed requests)
Cons: Still violates ToS if sharing is the intent
```

---

## 🎯 Recommendation

**The pool sharing model you described WILL NOT WORK with TorBox.**

You have three realistic options:

1. **Users provide their own TorBox keys** (Most viable)
   - No sharing, no ToS violation
   - Users pay for their own accounts
   - You provide the proxy/addon infrastructure

2. **Get an official partnership** (Best but difficult)
   - Contact TorBox about commercial agreements
   - Many services have partnerships allowing pooling
   - Provides legitimacy and stability

3. **Accept that accounts will be terminated** (Not recommended)
   - Try pooling, get caught, rotate accounts
   - Constant disruption for users
   - Bad customer experience

---

## 🚨 What You Need to Do

**Before proceeding, you need to**:

1. **Test with TorBox**
   - Set up pool key with 5-10 users
   - Run for 24 hours with diverse IPs
   - See if TorBox detects and flags it

2. **Contact TorBox**
   - Explain your use case
   - Ask if pool sharing is allowed
   - See if they offer commercial agreements

3. **Read ToS carefully**
   - TorBox ToS section on sharing
   - Check for any partnership/reseller options

4. **Plan fallback**
   - What if pooling gets blocked?
   - Can you switch to user-provided keys?
   - How will you notify users?

---

## Summary

**Your concern is 100% valid and critical**.

The URL-based detection isn't the main issue - the **concurrent access patterns from different IPs** are the real problem. This will be detected by TorBox's fraud systems automatically.

**This architecture only works if**:
1. You have a legitimate partnership/agreement with TorBox
2. You route through proxies to mask IP diversity (expensive)
3. You don't actually pool keys (defeats the purpose)

**Before building further, test this with TorBox and get explicit permission or you'll waste months of development on something that gets shut down on day one.**

---

**Status**: Critical Architecture Issue Identified  
**Recommendation**: Contact TorBox before proceeding  
**Action Required**: Feasibility testing before Phase 2

