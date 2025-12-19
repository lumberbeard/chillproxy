# Mandatory Prowlarr Torz Addon for PRO Users - Implementation Complete

**Date**: December 18, 2025  
**Status**: ✅ Complete

---

## Changes Made

### 1. **Removed from wizard_pro.json**
   - Removed hardcoded Prowlarr Torz addon from presets array
   - Template now only contains AIOMetadata addon

### 2. **Removed from Wizard Types**
   - Removed `'prowlarr'` from `BuiltInAddonIdSchema` 
   - Prowlarr is no longer an optional user selection in Step 2
   - File: `packages/wizard/shared/types/wizard.ts`

### 3. **Added Mandatory Prowlarr for PRO Users**
   - Updated `mergePresetsWithDefaults()` method in wizard-transform service
   - Now automatically injects Prowlarr Torz addon for ALL pro users
   - File: `packages/wizard/backend/services/wizard-transform.service.ts`

### 4. **Removed Optional Prowlarr Case Statement**
   - Deleted the case `'prowlarr'` from `transformStep2AddonsToPresets()` 
   - Prowlarr is no longer optional - it's automatic for pro users

---

## How It Works

### **Transformation Flow**

When a PRO user goes through the wizard:

1. **Step 1-3**: User configures filters, selects optional addons, picks catalogs
2. **Transformation**: When `transformToUserData()` is called with `subType: 'pro'`
3. **Preset Merging**: `mergePresetsWithDefaults()` is called with wizard data
4. **Prowlarr Injection**: 
   - Detects user is PRO (`subType === 'pro'`)
   - Gets user's UUID (their actual user ID)
   - Creates Prowlarr config with:
     - Prowlarr indexer URL: `http://localhost:9696/api/v2.0/indexers/all/results/torznab`
     - Prowlarr API key: `f963a60693dd49a08ff75188f9fc72d2` (TODO: make configurable)
     - Chillproxy store auth: User's UUID (for pool authentication)
   - Base64 encodes the config
   - Generates Chillproxy manifest URL with encoded config
   - Adds preset to the user's configuration

5. **Result**: User gets Prowlarr Torz addon with their UUID automatically included

---

## Code Implementation

### **In mergePresetsWithDefaults()**

```typescript
// For pro users, add Prowlarr Torz addon as mandatory indexing solution
const isPro = (wizardData as any)?.subType === 'pro';
if (isPro && wizardData?.uuid) {
  logger.info('💛 TORPOOL | Adding mandatory Prowlarr Torz addon for PRO user', { 
    uuid: wizardData.uuid 
  });

  // Build user-specific Prowlarr config with their UUID
  const prowlarrConfig = {
    indexers: [
      {
        url: 'http://localhost:9696/api/v2.0/indexers/all/results/torznab',
        apiKey: 'f963a60693dd49a08ff75188f9fc72d2',
      },
    ],
    stores: [
      {
        c: 'tb',
        t: '',
        auth: wizardData.uuid, // User's actual UUID for pool authentication
      },
    ],
  };

  // Base64 encode and create preset
  const prowlarrConfigJson = JSON.stringify(prowlarrConfig);
  const prowlarrConfigBase64 = Buffer.from(prowlarrConfigJson).toString('base64');

  allPresets.push({
    type: 'custom',
    instanceId: 'prowlarr-torz-mandatory',
    enabled: true,
    options: {
      name: 'Prowlarr Torz',
      manifestUrl: `http://localhost:8080/stremio/torz/${prowlarrConfigBase64}/manifest.json`,
      timeout: 20000,
      resources: [],
      mediaTypes: [],
      libraryAddon: false,
      formatPassthrough: false,
      resultPassthrough: false,
      forceToTop: false,
    },
  });
}
```

---

## Key Features

✅ **Mandatory for PRO Users**: All pro users get Prowlarr Torz automatically  
✅ **User-Specific**: Each user's config contains THEIR UUID  
✅ **Automatic**: No user selection needed - injected during transformation  
✅ **Clean**: No hardcoding in templates or defaults  
✅ **Logged**: Clear logging shows when addon is added  
✅ **Fallback**: BASIC users don't get Prowlarr (only pro users)

---

## TODO: Future Improvements

1. **Configurable Prowlarr URL**: Currently hardcoded to `http://localhost:9696`
   - Should be configurable per environment or per user

2. **Configurable Prowlarr API Key**: Currently hardcoded to `f963a60693dd49a08ff75188f9fc72d2`
   - Should be stored securely in environment variables or database

3. **User Preferences**: Consider allowing pro users to opt-out or customize Prowlarr settings

---

## Testing

To verify the changes work:

1. **Create a PRO user** through the wizard
2. **Check the generated UserData** - should include Prowlarr Torz preset
3. **Check the manifest** - should include Prowlarr addon with correct UUID
4. **Verify in Stremio** - should see "Prowlarr Torz" addon automatically

Example expected preset:
```json
{
  "type": "custom",
  "instanceId": "prowlarr-torz-mandatory",
  "enabled": true,
  "options": {
    "name": "Prowlarr Torz",
    "manifestUrl": "http://localhost:8080/stremio/torz/eyJpbmRleGVycyI6W3sidXJsIjoiaHR0cDovL2xvY2FsaG9zdDo5Njk2L2FwaS92Mi4wL2luZGV4ZXJzL2FsbC9yZXN1bHRzL3RvcnpuYWIiLCJhcGlLZXkiOiJmOTYzYTYwNjkzZGQ0OWEwOGZmNzUxODhmOWZjNzJkMiJ9XSwic3RvcmVzIjpbeyJjIjoidGIiLCJ0IjoiIiwiYXV0aCI6IjNiOTRjYjQ1LTNmOTktNDA2ZS05YzQwLWVjY2U2MWE0MDVjYyJ9XX0=/manifest.json",
    "timeout": 20000,
    "libraryAddon": false,
    "formatPassthrough": false,
    "resultPassthrough": false,
    "forceToTop": false
  }
}
```

---

## Summary

**Prowlarr Torz addon is now mandatory for all PRO users**, automatically injected during the wizard transformation with each user's unique UUID for pool authentication. No user selection needed - it's a built-in feature of the PRO plan.


