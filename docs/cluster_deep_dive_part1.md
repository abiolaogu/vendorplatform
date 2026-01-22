# Vendor & Artisans Platform
## Comprehensive Service Cluster Deep-Dive

---

# PART 1: PRIMARY CLUSTERS

---

## CLUSTER 1: CELEBRATIONS & LIFE EVENTS

### 1.1 Overview
The Celebrations cluster represents the highest-value, most complex service orchestration opportunity. A single wedding can involve 40+ distinct service categories, making this the ideal proving ground for adjacency-powered recommendations.

**Market Size (Nigeria):**
- Weddings: ~500,000/year × ₦3M average = ₦1.5T
- Funerals: ~300,000/year × ₦1.5M average = ₦450B
- Birthdays/Parties: ~2M events/year × ₦200K average = ₦400B
- **Total Addressable Market: ~₦2.4T annually**

### 1.2 User Journeys

#### Journey A: Traditional Wedding (₦5-10M Budget)

```
PHASE 1: ENGAGEMENT (T-365 to T-180 days)
┌─────────────────────────────────────────────────────────────────┐
│ TRIGGER: Engagement announcement                                │
├─────────────────────────────────────────────────────────────────┤
│ Emotional State: Excited, overwhelmed, unsure where to start    │
│ Primary Need: Understanding what's required and rough budget    │
├─────────────────────────────────────────────────────────────────┤
│ TOUCHPOINT 1: Platform Discovery                                │
│ • Bride searches "wedding planning Lagos"                       │
│ • Lands on EventGPT or wedding planning guide                   │
│ • Creates account with event type: "Traditional Wedding"        │
│                                                                 │
│ ADJACENCY TRIGGER: Event type detected → Generate checklist     │
│ RECOMMENDATIONS:                                                │
│ 1. Event Planners (if budget allows)                           │
│ 2. Venue exploration guide                                      │
│ 3. Budget calculator tool                                       │
│ 4. Timeline generator                                          │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│ TOUCHPOINT 2: Venue Selection (CRITICAL DECISION POINT)         │
├─────────────────────────────────────────────────────────────────┤
│ User browses venues, filters by:                                │
│ • Location (Lekki, VI, Ikeja)                                  │
│ • Capacity (200-300 guests)                                    │
│ • Indoor/Outdoor preference                                     │
│ • Date availability                                            │
│                                                                 │
│ ADJACENCY CASCADE (Venue → Everything):                         │
│ When user views venue, show:                                    │
│ ┌─────────────────────────────────────────────────────────┐    │
│ │ "Vendors who work great with [Venue Name]:"             │    │
│ │ • Caterers familiar with venue kitchen (Score: 0.95)    │    │
│ │ • Decorators who've transformed this space (0.92)       │    │
│ │ • Photographers who know best angles (0.90)             │    │
│ │ • Sound engineers familiar with acoustics (0.85)        │    │
│ └─────────────────────────────────────────────────────────┘    │
│                                                                 │
│ BUNDLE OPPORTUNITY:                                            │
│ "Book venue + catering together for 8% discount"               │
└─────────────────────────────────────────────────────────────────┘

PHASE 2: PLANNING (T-180 to T-60 days)
┌─────────────────────────────────────────────────────────────────┐
│ TOUCHPOINT 3: Catering Selection                                │
├─────────────────────────────────────────────────────────────────┤
│ Post-venue booking, user returns to platform                    │
│                                                                 │
│ CONTEXTUAL RECOMMENDATIONS:                                     │
│ "Based on your venue booking, you might need:"                  │
│ • Catering (95% of similar users booked) ← HIGHLIGHT           │
│ • Event Decoration (92%)                                        │
│ • Photography (90%)                                             │
│                                                                 │
│ When selecting caterer, cascade to:                             │
│ • Cake vendors (90% adjacency)                                  │
│ • Drinks/Bartending services (85%)                              │
│ • Waitstaff/Ushers (80%)                                       │
│ • Equipment rental - tables, chairs (75%)                       │
│                                                                 │
│ SMART SUGGESTION:                                               │
│ "Your caterer [Name] partners with [Cake Vendor] -              │
│  Book both for seamless coordination + 5% off"                  │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│ TOUCHPOINT 4: Visual Services Selection                         │
├─────────────────────────────────────────────────────────────────┤
│ User explores Photography services                              │
│                                                                 │
│ ADJACENCY CASCADE:                                              │
│ Photography → Videography (88% co-purchase)                     │
│ Photography → Makeup Artist (85% - bride needs look ready)      │
│ Photography → Hair Stylist (82%)                                │
│ Photography → Event Lighting (78% - affects photo quality)      │
│                                                                 │
│ CROSS-CLUSTER RECOMMENDATION:                                   │
│ "Photography chosen - Consider Bridal Styling package?"         │
│ → Links to Fashion & Personal Care cluster                      │
│                                                                 │
│ BUNDLE: "Complete Visual Package"                               │
│ Photography + Videography + Lighting = 12% savings              │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│ TOUCHPOINT 5: Bridal Preparation                                │
├─────────────────────────────────────────────────────────────────┤
│ Cross-cluster trigger: Wedding → Fashion cluster                │
│                                                                 │
│ BRIDAL STYLING CASCADE:                                         │
│ Event Fashion (Aso-Ebi, Wedding dress)                         │
│ ├── Tailors/Seamstresses (custom outfits)                      │
│ ├── Fabric Vendors (for Aso-Ebi coordination)                  │
│ ├── Makeup Artists (bridal makeup trial + day-of)              │
│ │   └── Skincare prep (recommended 3 months before)            │
│ ├── Hair Stylists (bridal hair)                                │
│ ├── Nail Technicians                                           │
│ ├── Henna Artists (for traditional ceremonies)                 │
│ ├── Jewelry/Accessories vendors                                │
│ └── Shoe makers/vendors                                        │
│                                                                 │
│ TIMELINE INTELLIGENCE:                                          │
│ "Book makeup trial 60 days before, final session 7 days before"│
└─────────────────────────────────────────────────────────────────┘

PHASE 3: FINALIZATION (T-60 to T-7 days)
┌─────────────────────────────────────────────────────────────────┐
│ TOUCHPOINT 6: Support Services                                  │
├─────────────────────────────────────────────────────────────────┤
│ Platform proactively surfaces remaining needs:                  │
│                                                                 │
│ PROGRESS DASHBOARD:                                             │
│ ✅ Venue (booked)                                               │
│ ✅ Catering (booked)                                            │
│ ✅ Photography (booked)                                         │
│ ⏳ MC/Host (recommended - 85% of weddings)                      │
│ ⏳ DJ/Entertainment (recommended - 90% of weddings)             │
│ ⏳ Event Security (recommended for 200+ guests)                 │
│ ⏳ Bridal Car (recommended)                                     │
│ ⚠️ Guest Transportation (consider for out-of-town guests)       │
│                                                                 │
│ LAST-MINUTE ADJACENCIES:                                        │
│ "7 days to your event - don't forget:"                          │
│ • Invitation printing (if not done)                            │
│ • Gift packaging services                                       │
│ • Emergency backup contacts for all vendors                     │
└─────────────────────────────────────────────────────────────────┘

PHASE 4: EVENT DAY (T-0)
┌─────────────────────────────────────────────────────────────────┐
│ TOUCHPOINT 7: Event Day Coordination                            │
├─────────────────────────────────────────────────────────────────┤
│ EVENT DAY COMMAND CENTER:                                       │
│ • Real-time vendor check-in tracking                           │
│ • Contact list for all booked vendors                          │
│ • Timeline view with vendor arrival times                      │
│ • Emergency support hotline                                    │
│                                                                 │
│ POST-EVENT TRIGGERS:                                            │
│ T+1: "How was your wedding? Rate your vendors"                 │
│ T+7: "Book honeymoon services?" → Travel cluster               │
│ T+30: "Thank you card printing services"                       │
│ T+365: "Anniversary coming up - celebrate again?"              │
└─────────────────────────────────────────────────────────────────┘
```

#### Journey B: Funeral/Memorial Service

```
TRIGGER: Death of family member (EMERGENCY CONTEXT)
┌─────────────────────────────────────────────────────────────────┐
│ EMOTIONAL STATE: Grief, overwhelm, urgency                      │
│ TIME PRESSURE: 3-7 days typical                                 │
│ DECISION MAKER: Usually one family member coordinating          │
├─────────────────────────────────────────────────────────────────┤
│ CRITICAL: Platform must be SENSITIVE and EFFICIENT              │
│ - Minimize decisions required                                   │
│ - Provide packages, not à la carte                             │
│ - 24/7 support availability                                    │
│ - Respectful, calm UI/UX                                       │
└─────────────────────────────────────────────────────────────────┘

IMMEDIATE NEEDS (Day 0-1):
├── Mortuary Services (FIRST CONTACT)
│   └── Adjacencies triggered immediately:
│       • Body transport services
│       • Embalming services
│       • Death certificate processing
│
├── When mortuary booked, surface:
│   • Casket/Coffin vendors
│   • Burial plot services (if not pre-purchased)
│   • Religious officiant booking
│
└── BUNDLE: "Immediate Care Package"
    Mortuary + Transport + Basic Casket + Certificate = Fixed price

PLANNING NEEDS (Day 1-3):
├── Venue for Service
│   └── Church, mosque, or event hall
│       Adjacencies: Sound system, seating, canopy rental
│
├── Program & Stationery
│   • Obituary writing services
│   • Program printing
│   • Memorial photo/video montage
│
├── Catering (for after-service)
│   └── Similar to event catering but somber packages
│
└── Family Support Services
    • Transportation for family members
    • Accommodation for travelers
    • Professional grief counseling referrals

POST-BURIAL NEEDS (Day 7+):
├── Headstone/Memorial engraving
├── Estate planning lawyers
├── Insurance claim assistance
└── Memorial anniversary reminders (T+365)
```

### 1.3 Technical Architecture for Celebrations Cluster

```
┌─────────────────────────────────────────────────────────────────┐
│                    CELEBRATIONS MICROSERVICE                     │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐            │
│  │   Event     │  │   Vendor    │  │  Checklist  │            │
│  │  Detection  │  │  Matching   │  │  Generator  │            │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘            │
│         │                │                │                    │
│         ▼                ▼                ▼                    │
│  ┌─────────────────────────────────────────────────┐          │
│  │            Event Orchestration Engine            │          │
│  │  • Timeline management                          │          │
│  │  • Budget allocation & tracking                 │          │
│  │  • Vendor coordination                          │          │
│  │  • Progress monitoring                          │          │
│  └─────────────────────────────────────────────────┘          │
│                          │                                     │
│         ┌────────────────┼────────────────┐                   │
│         ▼                ▼                ▼                   │
│  ┌───────────┐    ┌───────────┐    ┌───────────┐             │
│  │ Wedding   │    │ Funeral   │    │ Birthday  │             │
│  │ Module    │    │ Module    │    │ Module    │             │
│  └───────────┘    └───────────┘    └───────────┘             │
│                                                                │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    ADJACENCY RECOMMENDATION                      │
├─────────────────────────────────────────────────────────────────┤
│  Input: Current booking(s), Event type, Budget, Timeline        │
│                                                                 │
│  Process:                                                       │
│  1. Query adjacency graph for event context                     │
│  2. Filter by:                                                  │
│     - Already booked (exclude)                                  │
│     - Budget remaining (price filter)                           │
│     - Timeline fit (booking offset)                             │
│     - Location compatibility                                    │
│  3. Rank by:                                                    │
│     - Adjacency score                                           │
│     - Necessity score for event type                            │
│     - Vendor availability                                       │
│     - User preference signals                                   │
│  4. Diversify (ensure category variety)                         │
│                                                                 │
│  Output: Ranked list with explanations and bundle opportunities │
└─────────────────────────────────────────────────────────────────┘
```

### 1.4 Data Model Extensions for Celebrations

```sql
-- Wedding-specific details
CREATE TABLE wedding_details (
    project_id UUID PRIMARY KEY REFERENCES projects(id),
    wedding_type VARCHAR(50), -- 'traditional', 'white', 'court', 'destination'
    ceremony_style VARCHAR(50), -- 'religious', 'civil', 'cultural'
    bride_name VARCHAR(200),
    groom_name VARCHAR(200),
    wedding_hashtag VARCHAR(100),
    color_scheme JSONB, -- {"primary": "#FFD700", "secondary": "#FFFFFF"}
    aso_ebi_details JSONB, -- Fabric details for guests
    guest_list_size INTEGER,
    bridal_party_size INTEGER,
    dietary_requirements TEXT[],
    cultural_requirements TEXT[],
    has_traditional_ceremony BOOLEAN DEFAULT TRUE,
    has_white_wedding BOOLEAN DEFAULT TRUE,
    has_court_wedding BOOLEAN DEFAULT FALSE,
    reception_style VARCHAR(50) -- 'seated', 'cocktail', 'buffet'
);

-- Funeral-specific details
CREATE TABLE funeral_details (
    project_id UUID PRIMARY KEY REFERENCES projects(id),
    deceased_name VARCHAR(200) NOT NULL,
    relationship_to_planner VARCHAR(100),
    religion VARCHAR(50),
    burial_type VARCHAR(50), -- 'burial', 'cremation', 'sea_burial'
    service_type VARCHAR(50), -- 'church', 'mosque', 'graveside', 'memorial_only'
    lying_in_state BOOLEAN DEFAULT FALSE,
    wake_keeping BOOLEAN DEFAULT FALSE,
    cultural_rites TEXT[],
    obituary_text TEXT,
    tribute_video_needed BOOLEAN DEFAULT FALSE,
    expected_mourners INTEGER,
    family_contact_id UUID REFERENCES users(id)
);
```

---

## CLUSTER 2: HOME & PROPERTY SERVICES

### 2.1 Overview
Home services represent the highest-frequency transaction cluster with strong recurring revenue potential. Unlike celebrations (1-2 per lifetime), home services can generate 10-20 transactions per household annually.

**Market Size (Nigeria):**
- Home maintenance: ~10M households × ₦200K/year = ₦2T
- Moving/Relocation: ~500K moves × ₦300K average = ₦150B
- Renovation: ~200K projects × ₦2M average = ₦400B
- **Total Addressable Market: ~₦2.5T annually**

### 2.2 User Journeys

#### Journey A: Home Purchase & Setup

```
TRIGGER: New home purchase/rental
┌─────────────────────────────────────────────────────────────────┐
│ PHASE 1: PRE-MOVE INSPECTION (T-30 to T-14 days)               │
├─────────────────────────────────────────────────────────────────┤
│ User marks "Moving to new home" in app                          │
│                                                                 │
│ INITIAL ADJACENCY CASCADE:                                      │
│ "Before you move, consider inspecting:"                         │
│                                                                 │
│ Critical Inspections:                                           │
│ ├── Electrical inspection (safety check) - Score: 0.90         │
│ ├── Plumbing inspection (leak check) - Score: 0.88             │
│ ├── Pest inspection (termites, rodents) - Score: 0.85          │
│ └── Security assessment - Score: 0.80                          │
│                                                                 │
│ BUNDLE: "Pre-Move Inspection Package"                           │
│ Electrical + Plumbing + Pest + Security = ₦45,000 (Save 20%)   │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│ PHASE 2: PREPARATION (T-14 to T-3 days)                        │
├─────────────────────────────────────────────────────────────────┤
│ Based on inspection results, recommend:                         │
│                                                                 │
│ If electrical issues found:                                     │
│ → Electrician for repairs                                       │
│ → Smart home installation opportunity                           │
│                                                                 │
│ If plumbing issues found:                                       │
│ → Plumber for repairs                                           │
│ → Waterproofing services if bathroom/kitchen affected           │
│                                                                 │
│ If pest signs found:                                            │
│ → Fumigation BEFORE moving furniture in                         │
│ → Ongoing pest control contract offer                           │
│                                                                 │
│ Regardless of inspection:                                       │
│ → Deep cleaning service (95% co-purchase rate)                  │
│ → Painting touch-ups (70% opt for fresh paint)                  │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│ PHASE 3: MOVING DAY (T-0)                                       │
├─────────────────────────────────────────────────────────────────┤
│ PRIMARY SERVICE: Moving company                                 │
│                                                                 │
│ MOVING DAY ADJACENCIES:                                         │
│ ├── Packing materials/services                                  │
│ ├── Furniture disassembly/reassembly                           │
│ ├── Appliance disconnection/reconnection                        │
│ ├── Vehicle for personal transport                              │
│ └── Storage unit (if needed)                                   │
│                                                                 │
│ SAME-DAY SERVICES:                                              │
│ → Old home cleaning (for deposit return)                        │
│ → Key handover coordination                                     │
│                                                                 │
│ BUNDLE: "Complete Moving Day"                                   │
│ Movers + Packing + Assembly + Cleaning = ₦180,000 (Save 15%)   │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│ PHASE 4: NEW HOME SETUP (T+1 to T+30 days)                     │
├─────────────────────────────────────────────────────────────────┤
│ POST-MOVE ADJACENCY CASCADE:                                    │
│                                                                 │
│ Immediate Needs (Week 1):                                       │
│ ├── Furniture assembly (IKEA, etc.)                            │
│ ├── Appliance installation (AC, washing machine)               │
│ ├── Curtain/blind installation                                 │
│ ├── TV mounting                                                │
│ └── Internet/cable setup                                       │
│                                                                 │
│ Enhancement Needs (Week 2-4):                                   │
│ ├── Interior decoration consultation                           │
│ ├── Additional furniture (recommend local makers)              │
│ ├── Landscaping (if house with garden)                         │
│ ├── Security system installation                               │
│ └── Generator/inverter setup                                   │
│                                                                 │
│ Ongoing Services (Subscription opportunity):                    │
│ → Monthly cleaning service                                      │
│ → Quarterly pest control                                        │
│ → Annual maintenance package                                    │
└─────────────────────────────────────────────────────────────────┘
```

#### Journey B: Home Emergency Response

```
TRIGGER: Plumbing burst at 2 AM
┌─────────────────────────────────────────────────────────────────┐
│ EMERGENCY CONTEXT - SPEED IS CRITICAL                           │
├─────────────────────────────────────────────────────────────────┤
│ User opens app → Hits "Emergency" button                        │
│                                                                 │
│ STEP 1: Rapid Triage                                            │
│ "What's your emergency?"                                        │
│ [🚰 Plumbing] [⚡ Electrical] [🔒 Security] [🔥 Fire/Flood]      │
│                                                                 │
│ User selects: Plumbing                                          │
│ "Describe briefly": "Pipe burst, water everywhere"              │
│                                                                 │
│ STEP 2: Immediate Dispatch                                      │
│ System automatically:                                           │
│ 1. Identifies user location                                     │
│ 2. Finds nearest available emergency plumber                    │
│ 3. Shows ETA and rating                                         │
│ 4. One-tap booking                                              │
│                                                                 │
│ "Emergency plumber en route - ETA 45 minutes"                   │
│ "Meanwhile, turn off main water valve (usually near meter)"     │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│ POST-EMERGENCY ADJACENCIES (within 24-72 hours)                 │
├─────────────────────────────────────────────────────────────────┤
│ After plumbing emergency resolved:                              │
│                                                                 │
│ WATER DAMAGE CASCADE:                                           │
│ ├── Water damage restoration (0.95 - almost always needed)      │
│ │   └── Industrial fans, dehumidifiers, extraction              │
│ ├── Mold inspection/remediation (0.80 - if significant water)   │
│ ├── Flooring repair/replacement (0.70 - if wood/carpet)         │
│ ├── Painting (0.65 - water stains on walls)                    │
│ ├── Electrical inspection (0.60 - if water near outlets)       │
│ └── Furniture restoration (0.50 - if furniture affected)        │
│                                                                 │
│ INSURANCE ASSISTANCE:                                           │
│ → Help filing insurance claim                                   │
│ → Documentation service for damage                              │
│ → Contractor quotes for claim support                           │
│                                                                 │
│ PREVENTION:                                                     │
│ "Prevent future emergencies with:"                              │
│ → Annual plumbing inspection subscription                       │
│ → Water leak detection sensors                                  │
│ → Emergency response membership (priority dispatch)             │
└─────────────────────────────────────────────────────────────────┘
```

### 2.3 Technical Architecture for Home Services

```
┌─────────────────────────────────────────────────────────────────┐
│                    HOME SERVICES MICROSERVICE                    │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                   Property Registry                      │   │
│  │  • Property profiles (address, type, size, age)         │   │
│  │  • System inventory (AC units, plumbing age, etc.)      │   │
│  │  • Maintenance history                                  │   │
│  │  • Service subscriptions                                │   │
│  └─────────────────────────────────────────────────────────┘   │
│                              │                                  │
│              ┌───────────────┼───────────────┐                 │
│              ▼               ▼               ▼                 │
│  ┌───────────────┐ ┌───────────────┐ ┌───────────────┐        │
│  │   Emergency   │ │  Maintenance  │ │  Renovation   │        │
│  │   Response    │ │   Scheduler   │ │   Planner     │        │
│  │   Engine      │ │               │ │               │        │
│  └───────────────┘ └───────────────┘ └───────────────┘        │
│         │                 │                 │                  │
│         │    ┌────────────┴────────────┐   │                  │
│         │    ▼                         ▼   │                  │
│         │ ┌───────────────────────────────┐│                  │
│         │ │    Predictive Maintenance     ││                  │
│         │ │  • Seasonal prep reminders    ││                  │
│         │ │  • Equipment lifecycle alerts ││                  │
│         │ │  • Pattern-based predictions  ││                  │
│         │ └───────────────────────────────┘│                  │
│         │                                   │                  │
│         ▼                                   ▼                  │
│  ┌─────────────────────────────────────────────────────────┐  │
│  │              Vendor Dispatch & Routing                   │  │
│  │  • Real-time availability                               │  │
│  │  • Location-based matching                              │  │
│  │  • Skill-based routing                                  │  │
│  │  • Emergency priority queuing                           │  │
│  └─────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘

EMERGENCY RESPONSE FLOW:
┌──────┐    ┌──────────┐    ┌───────────┐    ┌──────────┐
│ User │───▶│ Classify │───▶│  Find     │───▶│ Dispatch │
│ SOS  │    │ Emergency│    │  Nearest  │    │ & Track  │
└──────┘    └──────────┘    │  Vendor   │    └──────────┘
                            └───────────┘
                                  │
                            ┌─────┴─────┐
                            ▼           ▼
                      ┌─────────┐ ┌─────────┐
                      │ Primary │ │ Backup  │
                      │ Vendor  │ │ Vendor  │
                      └─────────┘ └─────────┘
```

### 2.4 Home Service Subscription Models

```
┌─────────────────────────────────────────────────────────────────┐
│                  SUBSCRIPTION TIERS                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  BASIC (₦10,000/month)                                         │
│  ├── Priority booking (same-day for non-emergency)             │
│  ├── 10% discount on all services                              │
│  ├── Quarterly maintenance reminders                           │
│  └── Email support                                             │
│                                                                 │
│  STANDARD (₦25,000/month)                                      │
│  ├── Everything in Basic                                       │
│  ├── 1 free emergency callout per quarter                      │
│  ├── Bi-annual deep cleaning included                          │
│  ├── Annual AC servicing (up to 2 units)                       │
│  ├── Quarterly pest control                                    │
│  └── Phone support                                             │
│                                                                 │
│  PREMIUM (₦50,000/month)                                       │
│  ├── Everything in Standard                                    │
│  ├── Unlimited emergency callouts (2-hour response SLA)        │
│  ├── Monthly cleaning included                                 │
│  ├── All appliance servicing covered                           │
│  ├── Annual electrical & plumbing inspection                   │
│  ├── Dedicated account manager                                 │
│  ├── 20% discount on renovation projects                       │
│  └── 24/7 support hotline                                      │
│                                                                 │
│  PROPERTY MANAGER (Custom pricing)                              │
│  ├── Multi-property dashboard                                  │
│  ├── Tenant request management                                 │
│  ├── Vendor performance analytics                              │
│  ├── Budget tracking & reporting                               │
│  └── API integration                                           │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## CLUSTER 3: TRAVEL & MOBILITY

### 3.1 Overview
Travel services have clear, linear adjacency chains but significant cross-cluster connections (e.g., travel for weddings, business trips leading to event catering needs).

**Market Size (Nigeria):**
- Domestic air travel: ~10M trips × ₦100K average = ₦1T
- International travel: ~2M trips × ₦500K average = ₦1T
- Ground transportation: ~50M trips × ₦5K average = ₦250B
- **Total Addressable Market: ~₦2.25T annually**

### 3.2 User Journey: International Travel

```
TRIGGER: User books/mentions international trip
┌─────────────────────────────────────────────────────────────────┐
│ PHASE 1: TRAVEL PLANNING (T-90 to T-30 days)                   │
├─────────────────────────────────────────────────────────────────┤
│ User indicates: "Traveling to London in 2 months"               │
│                                                                 │
│ IMMEDIATE ADJACENCY CHECK:                                      │
│ "For UK travel, you'll need:"                                   │
│                                                                 │
│ VISA & DOCUMENTATION (if applicable):                           │
│ ├── Visa processing service (0.95 for countries requiring)      │
│ │   └── Document preparation, appointment booking               │
│ ├── Passport renewal (check expiry - 6 month rule)             │
│ └── Travel insurance (0.85 - highly recommended)                │
│                                                                 │
│ BUNDLE: "Travel Ready Package"                                  │
│ Visa assistance + Travel insurance + Airport transfer           │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│ PHASE 2: PRE-DEPARTURE (T-30 to T-7 days)                      │
├─────────────────────────────────────────────────────────────────┤
│ PREPARATION ADJACENCIES:                                        │
│                                                                 │
│ Financial Prep:                                                 │
│ ├── Currency exchange (0.90)                                   │
│ │   └── "Get competitive rates for GBP"                        │
│ └── International card activation reminder                      │
│                                                                 │
│ Packing & Gear:                                                 │
│ ├── Luggage purchase (if flagged as needed)                    │
│ ├── Travel adapters, accessories                               │
│ └── Weather-appropriate clothing (link to Fashion cluster)      │
│                                                                 │
│ Destination Services (pre-book):                                │
│ ├── Airport pickup at destination (0.85)                       │
│ ├── Hotel/Accommodation (if not booked)                        │
│ ├── Car rental                                                 │
│ └── Local SIM card delivery to hotel                           │
│                                                                 │
│ Home Care (while away):                                         │
│ → House sitting service                                         │
│ → Pet care (link to Pet cluster)                               │
│ → Plant watering service                                        │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│ PHASE 3: DEPARTURE DAY (T-0)                                    │
├─────────────────────────────────────────────────────────────────┤
│ DEPARTURE ADJACENCIES:                                          │
│                                                                 │
│ To Airport:                                                     │
│ ├── Airport transfer booking (0.95)                            │
│ │   └── Taxi, ride-hail, or chauffeur service                  │
│ ├── Porter service at airport                                  │
│ └── Fast-track immigration (if available)                      │
│                                                                 │
│ At Airport:                                                     │
│ ├── Lounge access (0.70 for long-haul)                        │
│ ├── Currency exchange (last-minute)                            │
│ └── Duty-free shopping reminders                               │
│                                                                 │
│ SMART NOTIFICATIONS:                                            │
│ "Your flight is in 6 hours. Airport transfer booked for 3pm."  │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│ PHASE 4: AT DESTINATION                                         │
├─────────────────────────────────────────────────────────────────┤
│ DESTINATION ADJACENCIES:                                        │
│                                                                 │
│ On Arrival:                                                     │
│ ├── Airport pickup confirmation                                │
│ ├── Local SIM activation                                       │
│ └── Hotel check-in details                                     │
│                                                                 │
│ During Stay:                                                    │
│ ├── Tour guides & experiences (0.80 for tourists)              │
│ ├── Restaurant reservations                                    │
│ ├── Chauffeur for meetings (business travelers)                │
│ └── Event tickets, attractions                                 │
│                                                                 │
│ CROSS-CLUSTER: If traveling for event                          │
│ → Link to Celebrations cluster for venue, catering at dest.    │
│                                                                 │
│ CROSS-CLUSTER: If business trip                                │
│ → Link to Business cluster for meeting rooms, printing         │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│ PHASE 5: RETURN (T+X)                                           │
├─────────────────────────────────────────────────────────────────┤
│ RETURN ADJACENCIES:                                             │
│                                                                 │
│ Pre-Return:                                                     │
│ ├── Return airport transfer at destination                     │
│ ├── Excess baggage shipping (if needed)                        │
│ └── Home airport pickup booking                                │
│                                                                 │
│ Post-Arrival:                                                   │
│ ├── Jet lag recovery (spa, wellness)                           │
│ ├── Laundry/dry cleaning (travel clothes)                      │
│ ├── Photo printing/album (vacation memories)                   │
│ └── "Plan your next trip?" recommendation                      │
│                                                                 │
│ FEEDBACK & LOYALTY:                                             │
│ → Rate travel services used                                     │
│ → Earn loyalty points                                           │
│ → Early access to travel deals                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## CLUSTER 4: FOOD & HOSPITALITY (HORECA)

### 4.1 Overview
HORECA services have both B2C (private chefs, meal prep) and B2B (restaurant supply, equipment) components with strong adjacencies to Celebrations and Business clusters.

**Market Size (Nigeria):**
- Restaurant/Food service: ₦3T industry
- Catering services: ₦500B
- Food delivery: ₦200B
- **Platform-addressable: ~₦500B annually**

### 4.2 User Journey: Restaurant Launch

```
TRIGGER: Entrepreneur decides to open restaurant
┌─────────────────────────────────────────────────────────────────┐
│ PHASE 1: PLANNING & SETUP (T-180 to T-90 days)                 │
├─────────────────────────────────────────────────────────────────┤
│ User selects: "Starting a food business"                        │
│                                                                 │
│ BUSINESS SETUP ADJACENCIES:                                     │
│ ├── Business registration (link to Business cluster)           │
│ │   └── CAC registration, food business permits                │
│ ├── Location scouting & lease negotiation                      │
│ ├── Restaurant consultant (concept, menu development)          │
│ └── Business plan assistance                                   │
│                                                                 │
│ LICENSING (CRITICAL):                                           │
│ ├── NAFDAC registration (for packaged foods)                   │
│ ├── State health department approval                           │
│ ├── Fire safety certificate                                    │
│ └── Environmental compliance                                   │
│                                                                 │
│ BUNDLE: "Restaurant Startup Legal Package"                      │
│ Business reg + Health permits + NAFDAC + Fire safety           │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│ PHASE 2: BUILD-OUT (T-90 to T-30 days)                         │
├─────────────────────────────────────────────────────────────────┤
│ INFRASTRUCTURE ADJACENCIES:                                     │
│                                                                 │
│ Interior & Design:                                              │
│ ├── Restaurant interior designer (0.90)                        │
│ ├── Kitchen equipment vendor (0.95 - essential)                │
│ │   └── Ovens, refrigerators, prep stations                   │
│ ├── Furniture (tables, chairs, bar stools)                     │
│ ├── Signage & branding (exterior, menu boards)                 │
│ └── POS system & software                                      │
│                                                                 │
│ Utilities & Safety:                                             │
│ ├── Commercial electrical work                                  │
│ ├── Plumbing (kitchen-grade)                                   │
│ ├── HVAC / ventilation (critical for kitchen)                  │
│ ├── Fire suppression system                                    │
│ └── Security system                                            │
│                                                                 │
│ CROSS-CLUSTER: Links to Home/Property for construction         │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│ PHASE 3: STAFFING & SUPPLIES (T-30 to T-7 days)                │
├─────────────────────────────────────────────────────────────────┤
│ STAFFING ADJACENCIES:                                           │
│ ├── Chef recruitment (head chef, sous chef)                    │
│ ├── Kitchen staff (line cooks, prep, dishwashers)              │
│ ├── Front of house (waitstaff, hosts, bartenders)              │
│ ├── Staff training services                                    │
│ └── Uniform supplier                                           │
│                                                                 │
│ SUPPLY CHAIN:                                                   │
│ ├── Food suppliers (produce, meat, seafood)                    │
│ ├── Beverage distributors                                      │
│ ├── Dry goods suppliers                                        │
│ ├── Cleaning supplies                                          │
│ └── Disposables (if applicable)                                │
│                                                                 │
│ ONGOING SERVICES (Set up contracts):                            │
│ ├── Pest control (mandatory for food businesses)               │
│ ├── Grease trap cleaning                                       │
│ ├── Linen service (tablecloths, napkins)                       │
│ └── Equipment maintenance                                       │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│ PHASE 4: LAUNCH & MARKETING (T-7 to T+30 days)                 │
├─────────────────────────────────────────────────────────────────┤
│ LAUNCH MARKETING ADJACENCIES:                                   │
│                                                                 │
│ Content Creation:                                               │
│ ├── Food photographer (0.90 - essential for Instagram)         │
│ ├── Menu designer (print & digital)                            │
│ ├── Videographer (launch video, social content)                │
│ └── Copywriter (menu descriptions, social captions)            │
│                                                                 │
│ Digital Presence:                                               │
│ ├── Website development                                        │
│ ├── Social media management                                    │
│ ├── Influencer partnerships (food bloggers)                    │
│ └── Online ordering integration                                │
│                                                                 │
│ Launch Event:                                                   │
│ ├── Event planner (soft opening)                               │
│ ├── PR agency (media invites)                                  │
│ └── Photographer for opening                                   │
│                                                                 │
│ CROSS-CLUSTER: Links to Creative & Business clusters           │
└─────────────────────────────────────────────────────────────────┘
```

### 4.3 User Journey: Private Chef Booking

```
TRIGGER: User wants intimate dinner party at home
┌─────────────────────────────────────────────────────────────────┐
│ SCENARIO: "Dinner party for 8 at my home this Saturday"         │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│ CORE SERVICE: Private Chef                                      │
│ • Browse by cuisine type (Nigerian, Continental, Asian)         │
│ • Filter by dietary (halal, vegetarian, allergies)             │
│ • View sample menus and pricing                                │
│                                                                 │
│ ADJACENCY CASCADE (once chef selected):                         │
│                                                                 │
│ Meal Enhancement:                                               │
│ ├── Sommelier / Wine pairing (0.75)                            │
│ ├── Bartender / Cocktails (0.70)                               │
│ ├── Dessert specialist (0.65)                                  │
│ └── Specialty ingredient sourcing (0.60)                       │
│                                                                 │
│ Service Staff:                                                  │
│ ├── Server / Waitstaff (0.80)                                  │
│ └── Cleanup crew (0.85)                                        │
│                                                                 │
│ Ambiance:                                                       │
│ ├── Table setting / Dinnerware rental (0.70)                   │
│ ├── Floral arrangements (0.65)                                 │
│ ├── Background music / Playlist curation (0.50)                │
│ └── Candles / Mood lighting (0.45)                             │
│                                                                 │
│ BUNDLE: "Complete Dinner Party"                                 │
│ Chef + Server + Wine + Flowers + Cleanup = 15% savings          │
│                                                                 │
│ POST-EVENT:                                                     │
│ → Rate chef and services                                        │
│ → "Book again" with saved preferences                          │
│ → Refer to friends (referral bonus)                            │
└─────────────────────────────────────────────────────────────────┘
```

---

*Continued in Part 2: Fashion, Business, Education, Health clusters*
*Continued in Part 3: Automotive, Creative, Property, Energy, Security, Pet clusters*
