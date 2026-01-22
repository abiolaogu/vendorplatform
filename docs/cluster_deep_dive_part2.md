# Vendor & Artisans Platform
## Comprehensive Service Cluster Deep-Dive

---

# PART 2: SECONDARY CLUSTERS

---

## CLUSTER 5: FASHION & PERSONAL CARE

### 5.1 Overview
Fashion services are deeply interconnected with Celebrations (bridal styling) and have strong recurring potential through personal grooming services.

**Market Size (Nigeria):**
- Personal grooming: ~20M urban consumers × ₦150K/year = ₦3T
- Fashion/Tailoring: ~₦500B
- Spa/Wellness: ~₦200B
- **Platform-addressable: ~₦500B annually**

### 5.2 User Journey: Complete Personal Makeover

```
TRIGGER: Major life change (new job, divorce, milestone birthday)
┌─────────────────────────────────────────────────────────────────┐
│ CONTEXT: User wants to "reinvent themselves"                    │
├─────────────────────────────────────────────────────────────────┤
│ User searches: "complete makeover Lagos"                        │
│                                                                 │
│ INTENT DETECTION:                                               │
│ → Signals: life change, confidence, transformation              │
│ → Recommended starting point: Personal Stylist consultation     │
│                                                                 │
│ PHASE 1: CONSULTATION                                           │
│ Personal Stylist (hub service):                                 │
│ ├── Style assessment                                           │
│ ├── Color analysis                                             │
│ ├── Wardrobe audit                                             │
│ └── Shopping plan                                              │
│                                                                 │
│ ADJACENCY CASCADE from Stylist:                                 │
│ "Your stylist recommends completing your transformation with:"  │
│                                                                 │
│ Hair & Grooming:                                                │
│ ├── Hair stylist / New haircut & color (0.92)                  │
│ ├── Barber (for men) (0.90)                                    │
│ ├── Skincare specialist / Facial treatments (0.85)             │
│ └── Nail technician (0.80)                                     │
│                                                                 │
│ Wardrobe Building:                                              │
│ ├── Tailor for custom pieces (0.88)                            │
│ ├── Fabric vendor (for custom outfits) (0.75)                  │
│ ├── Shoe maker/vendor (0.70)                                   │
│ └── Accessories/Jewelry (0.65)                                 │
│                                                                 │
│ Fitness & Wellness:                                             │
│ ├── Personal trainer (0.70)                                    │
│ ├── Nutritionist (0.65)                                        │
│ └── Spa day / Massage (0.75)                                   │
│                                                                 │
│ Professional Image:                                             │
│ ├── Professional photographer (new headshots) (0.80)           │
│ ├── LinkedIn profile expert (0.60)                             │
│ └── Image consultant (0.55)                                    │
└─────────────────────────────────────────────────────────────────┘

BUNDLE OPTIONS:
┌─────────────────────────────────────────────────────────────────┐
│ "Quick Refresh" - ₦150,000                                      │
│ Hair + Facial + Nails + Mini wardrobe consultation              │
├─────────────────────────────────────────────────────────────────┤
│ "Complete Transformation" - ₦500,000                            │
│ Stylist + Hair + Skincare + Tailor (2 outfits) + Photoshoot    │
├─────────────────────────────────────────────────────────────────┤
│ "Executive Package" - ₦1,000,000                                │
│ Everything above + Personal trainer (3 months) + Nutritionist   │
│ + 4 custom outfits + Quarterly maintenance sessions            │
└─────────────────────────────────────────────────────────────────┘
```

### 5.3 User Journey: Bridal Styling (Cross-Cluster)

```
TRIGGER: From Celebrations cluster - Wedding planning
┌─────────────────────────────────────────────────────────────────┐
│ CROSS-CLUSTER HANDOFF                                           │
│ User is planning wedding → "Bridal styling" category triggered  │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│ TIMELINE-BASED RECOMMENDATIONS:                                 │
│                                                                 │
│ T-90 DAYS (3 months before):                                    │
│ ├── Skincare specialist - Start skin prep regimen              │
│ │   └── Monthly facials leading to wedding                     │
│ ├── Fitness/Personal trainer - Wedding body goals              │
│ ├── Hair care - Treatment plan for healthy hair                │
│ └── Fabric vendor - Start Aso-Ebi coordination                 │
│                                                                 │
│ T-60 DAYS (2 months before):                                    │
│ ├── Tailor - Final fitting appointments                        │
│ ├── Makeup artist - Trial session                              │
│ ├── Hair stylist - Trial session                               │
│ └── Henna artist - If traditional ceremony                     │
│                                                                 │
│ T-30 DAYS (1 month before):                                     │
│ ├── Nail technician - Book wedding day appointment             │
│ ├── Spa - Pre-wedding relaxation package                       │
│ └── Jewelry finalization                                       │
│                                                                 │
│ T-7 DAYS (Week before):                                         │
│ ├── Final facial                                               │
│ ├── Hair treatment                                             │
│ ├── Nail prep                                                  │
│ └── Henna application (2-3 days before)                        │
│                                                                 │
│ T-1 DAY (Day before):                                           │
│ ├── Spa relaxation                                             │
│ └── Early sleep (wellness reminder)                            │
│                                                                 │
│ T-0 (Wedding Day):                                              │
│ ├── Hair styling (morning)                                     │
│ ├── Makeup application                                         │
│ ├── Nail touch-up                                              │
│ ├── Dressing assistance                                        │
│ └── Throughout-day touch-ups                                   │
│                                                                 │
│ POST-WEDDING:                                                   │
│ → Recovery spa day                                              │
│ → "Maintain your glow" skincare subscription                   │
└─────────────────────────────────────────────────────────────────┘
```

---

## CLUSTER 6: BUSINESS & CORPORATE

### 6.1 Overview
Business services drive B2B revenue and create sticky, recurring relationships. Strong adjacencies to all other clusters as businesses need various services.

**Market Size (Nigeria):**
- Business registration/legal: ~200K new businesses × ₦200K = ₦40B
- Corporate events: ₦300B
- Office services: ₦500B
- **Platform-addressable: ~₦300B annually**

### 6.2 User Journey: Starting a Business

```
TRIGGER: Entrepreneur decides to formalize business
┌─────────────────────────────────────────────────────────────────┐
│ User indicates: "Register a new company"                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│ PHASE 1: LEGAL FOUNDATION                                       │
│ PRIMARY: Business Registration Agent                            │
│                                                                 │
│ IMMEDIATE ADJACENCIES:                                          │
│ ├── CAC registration service (0.98 - almost always)            │
│ │   └── Name search, document prep, filing                     │
│ ├── Business lawyer (0.85 - for complex structures)            │
│ │   └── Shareholder agreements, contracts                      │
│ ├── Accountant (0.90)                                          │
│ │   └── Tax registration, financial setup                      │
│ └── Corporate bank account opening assistance (0.92)           │
│                                                                 │
│ BUNDLE: "Business Starter Pack" - ₦150,000                      │
│ CAC Registration + Tax ID + Bank Account Setup + Basic Legal    │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│ PHASE 2: BRANDING & IDENTITY                                    │
├─────────────────────────────────────────────────────────────────┤
│ Once registered, trigger branding sequence:                     │
│                                                                 │
│ Visual Identity:                                                │
│ ├── Logo designer (0.90)                                       │
│ ├── Brand strategist (0.75)                                    │
│ ├── Stationery design (business cards, letterhead) (0.85)      │
│ └── Stationery printing (0.82)                                 │
│                                                                 │
│ Digital Presence:                                               │
│ ├── Website developer (0.85)                                   │
│ ├── Domain & hosting setup (0.80)                              │
│ ├── Email setup (Google Workspace, etc.) (0.75)                │
│ └── Social media setup (0.70)                                  │
│                                                                 │
│ BUNDLE: "Brand Launch Package" - ₦300,000                       │
│ Logo + Brand guide + Website + Business cards + Social setup    │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│ PHASE 3: OPERATIONS SETUP                                       │
├─────────────────────────────────────────────────────────────────┤
│ If physical office needed:                                      │
│                                                                 │
│ Office Space (Cross-cluster to Property):                       │
│ ├── Office space finder / Real estate agent                    │
│ ├── Co-working space membership                                │
│ └── Virtual office service                                     │
│                                                                 │
│ Office Setup:                                                   │
│ ├── Office furniture (0.90)                                    │
│ ├── IT equipment & setup (0.88)                                │
│ │   └── Computers, printers, network                          │
│ ├── Office cleaning service (0.80)                             │
│ └── Security system (0.70)                                     │
│                                                                 │
│ Staffing:                                                       │
│ ├── HR consultant / Recruiter (0.75)                           │
│ ├── Payroll service setup (0.70)                               │
│ └── Employee benefits broker (0.60)                            │
│                                                                 │
│ Insurance:                                                      │
│ ├── Business liability insurance (0.85)                        │
│ ├── Property insurance (0.70)                                  │
│ └── Employee health insurance (0.65)                           │
└─────────────────────────────────────────────────────────────────┘
```

### 6.3 User Journey: Corporate Event

```
TRIGGER: Company needs to host conference/product launch
┌─────────────────────────────────────────────────────────────────┐
│ EVENT TYPE: Product Launch (200 attendees)                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│ VENUE & LOGISTICS:                                              │
│ ├── Conference venue (0.98)                                    │
│ │   └── Hotels, convention centers, unique spaces              │
│ ├── AV equipment rental (0.95)                                 │
│ │   └── Projectors, screens, microphones, staging             │
│ ├── Event furniture rental (0.88)                              │
│ └── Registration system / Badge printing (0.85)                │
│                                                                 │
│ CATERING & HOSPITALITY:                                         │
│ ├── Corporate catering (0.95)                                  │
│ │   └── Coffee breaks, lunch, cocktails                       │
│ ├── Bartending service (0.70)                                  │
│ └── Waitstaff (0.80)                                          │
│                                                                 │
│ CONTENT & PRODUCTION:                                           │
│ ├── Event photographer (0.90)                                  │
│ ├── Videographer (0.85)                                        │
│ │   └── Live streaming, post-event video                      │
│ ├── Graphic designer (event materials) (0.80)                  │
│ ├── Presentation designer (0.75)                               │
│ └── Transcription / Note-taking service (0.60)                 │
│                                                                 │
│ MARKETING & PR:                                                 │
│ ├── PR agency (media invitations) (0.75)                       │
│ ├── Social media coverage (0.80)                               │
│ └── Influencer attendance (0.60)                               │
│                                                                 │
│ SUPPORT SERVICES:                                               │
│ ├── Event security (0.70)                                      │
│ ├── Parking management (0.60)                                  │
│ ├── Transport for VIPs (0.65)                                  │
│ └── Interpretation services (if international) (0.50)          │
│                                                                 │
│ BUNDLE: "Complete Corporate Event"                              │
│ Venue + AV + Catering + Photo/Video + Registration = 12% off   │
└─────────────────────────────────────────────────────────────────┘
```

---

## CLUSTER 7: EDUCATION & LEARNING

### 7.1 Overview
Education services span B2C (tutoring, test prep) and B2B (school supplies, institutional services) with strong seasonal patterns.

**Market Size (Nigeria):**
- Private tutoring: ~5M students × ₦300K/year = ₦1.5T
- Study abroad: ~50K students × ₦5M = ₦250B
- School supplies/services: ₦500B
- **Platform-addressable: ~₦200B annually**

### 7.2 User Journey: Study Abroad Preparation

```
TRIGGER: Student/Parent begins study abroad process
┌─────────────────────────────────────────────────────────────────┐
│ PHASE 1: RESEARCH & TESTING (T-18 to T-12 months)              │
├─────────────────────────────────────────────────────────────────┤
│ User indicates: "Want to study in UK/USA/Canada"                │
│                                                                 │
│ INITIAL ADJACENCIES:                                            │
│                                                                 │
│ Consultation:                                                   │
│ ├── Study abroad consultant (0.95)                             │
│ │   └── University selection, country advice                  │
│ └── Career counselor (for program selection) (0.70)            │
│                                                                 │
│ Test Preparation:                                               │
│ ├── IELTS/TOEFL prep classes (0.92)                           │
│ ├── SAT/ACT prep (for undergrad USA) (0.85)                   │
│ ├── GRE/GMAT prep (for graduate) (0.80)                       │
│ └── Test center booking assistance (0.75)                      │
│                                                                 │
│ BUNDLE: "Test Ready Package"                                    │
│ Consultant + IELTS prep (3 months) + Practice tests = ₦400K    │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│ PHASE 2: APPLICATION (T-12 to T-6 months)                      │
├─────────────────────────────────────────────────────────────────┤
│ APPLICATION SUPPORT:                                            │
│ ├── Statement of Purpose editor/writer (0.88)                  │
│ ├── Recommendation letter coordination (0.75)                  │
│ ├── Transcript processing (0.80)                               │
│ └── Application fee payment assistance (0.70)                  │
│                                                                 │
│ DOCUMENT SERVICES:                                              │
│ ├── Document attestation (0.85)                                │
│ ├── Translation services (if needed) (0.70)                    │
│ └── Notarization (0.75)                                        │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│ PHASE 3: PRE-DEPARTURE (T-6 to T-1 month)                      │
├─────────────────────────────────────────────────────────────────┤
│ VISA & TRAVEL (Cross-cluster to Travel):                        │
│ ├── Student visa processing (0.98)                             │
│ ├── Travel insurance (0.90)                                    │
│ ├── Flight booking (0.95)                                      │
│ ├── Forex / International banking setup (0.88)                 │
│ └── Luggage shopping (0.75)                                    │
│                                                                 │
│ ACCOMMODATION:                                                  │
│ ├── University housing assistance (0.85)                       │
│ ├── Private accommodation finder (0.80)                        │
│ └── Homestay arrangement (0.70)                                │
│                                                                 │
│ PRE-DEPARTURE PREP:                                             │
│ ├── Cultural orientation session (0.70)                        │
│ ├── Pre-departure health check (0.75)                          │
│ │   └── Vaccinations, medical clearance                       │
│ └── Packing consultation (0.50)                                │
│                                                                 │
│ BUNDLE: "Departure Ready"                                       │
│ Visa + Insurance + Flight + Accommodation assist = ₦800K       │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│ PHASE 4: ARRIVAL & SETTLING (T+0 to T+30 days)                 │
├─────────────────────────────────────────────────────────────────┤
│ ARRIVAL SERVICES:                                               │
│ ├── Airport pickup (0.90)                                      │
│ ├── Temporary accommodation (if needed) (0.80)                 │
│ ├── Local SIM card (0.85)                                      │
│ └── Bank account opening assistance (0.82)                     │
│                                                                 │
│ SETTLING IN:                                                    │
│ ├── Furniture shopping/rental (0.75)                           │
│ ├── Essential supplies shopping (0.70)                         │
│ ├── Student community connection (0.65)                        │
│ └── Local orientation guide (0.60)                             │
│                                                                 │
│ ONGOING SUPPORT:                                                │
│ → Academic tutoring at destination                              │
│ → Mental health support (student counseling)                    │
│ → Nigerian community connections                                │
│ → Holiday travel arrangements (cross-cluster)                   │
└─────────────────────────────────────────────────────────────────┘
```

---

## CLUSTER 8: HEALTH & WELLNESS

### 8.1 Overview
Health services require careful handling due to sensitivity. Strong adjacencies exist for recovery scenarios and ongoing wellness journeys.

**Market Size (Nigeria):**
- Home healthcare: ₦200B
- Fitness & wellness: ₦300B
- Alternative medicine/wellness: ₦100B
- **Platform-addressable: ~₦150B annually**

### 8.2 User Journey: Post-Surgery Recovery

```
TRIGGER: User scheduled for/recovering from surgery
┌─────────────────────────────────────────────────────────────────┐
│ CONTEXT: Sensitive health situation requiring care              │
│ APPROACH: Supportive, not pushy; focus on genuine needs        │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│ PRE-SURGERY (T-14 to T-0):                                      │
│ ├── Second opinion consultation (0.60)                         │
│ ├── Pre-op health assessment (0.75)                            │
│ ├── Medical supplies (post-op needs) (0.85)                    │
│ └── Home preparation consultation (0.70)                       │
│     └── Remove trip hazards, prepare recovery space            │
│                                                                 │
│ IMMEDIATE POST-SURGERY (T+0 to T+7):                            │
│ PRIMARY: Home nursing care (0.95)                               │
│                                                                 │
│ CASCADING ADJACENCIES:                                          │
│ ├── Pharmacy delivery (0.95 - medication management)           │
│ ├── Medical equipment rental (0.85)                            │
│ │   └── Hospital bed, wheelchair, walker, oxygen               │
│ ├── Meal prep service (0.80)                                   │
│ │   └── Nutritious, recovery-appropriate meals                │
│ └── Home modification (0.60)                                   │
│     └── Grab bars, ramps if needed                            │
│                                                                 │
│ RECOVERY PHASE (T+7 to T+30):                                   │
│ ├── Physiotherapy (0.90 for orthopedic surgeries)              │
│ ├── Occupational therapy (0.70)                                │
│ ├── Wound care specialist (0.80)                               │
│ ├── Lab tests at home (follow-up) (0.85)                       │
│ └── Telemedicine follow-ups (0.75)                             │
│                                                                 │
│ EMOTIONAL SUPPORT:                                              │
│ ├── Counseling / Mental health support (0.60)                  │
│ └── Support group connections (0.50)                           │
│                                                                 │
│ BUNDLE: "Complete Recovery Care"                                │
│ Nursing (7 days) + Physio (4 sessions) + Meal prep + Equipment │
│ Starting at ₦350,000                                            │
└─────────────────────────────────────────────────────────────────┘
```

### 8.3 User Journey: Fitness Transformation

```
TRIGGER: New Year resolution / Health wake-up call
┌─────────────────────────────────────────────────────────────────┐
│ User goal: "Lose weight and get fit"                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│ ASSESSMENT PHASE:                                               │
│ ├── Fitness assessment (0.85)                                  │
│ ├── Health screening (0.75)                                    │
│ │   └── Basic labs, BMI, body composition                     │
│ └── Nutritionist consultation (0.80)                           │
│                                                                 │
│ CORE FITNESS SERVICES:                                          │
│ ├── Personal trainer (0.92)                                    │
│ │   └── 1-on-1 or small group                                 │
│ ├── Gym membership (0.85)                                      │
│ ├── Online fitness classes (0.70)                              │
│ └── Home gym equipment (0.65)                                  │
│                                                                 │
│ NUTRITION SUPPORT:                                              │
│ ├── Meal prep service (0.85)                                   │
│ │   └── Calorie-controlled, macro-balanced                    │
│ ├── Nutritionist (ongoing) (0.80)                              │
│ ├── Grocery delivery (healthy foods) (0.70)                    │
│ └── Supplement consultation (0.60)                             │
│                                                                 │
│ WELLNESS COMPLEMENT:                                            │
│ ├── Yoga instructor (0.75)                                     │
│ ├── Meditation guide (0.65)                                    │
│ ├── Sports massage (0.70)                                      │
│ └── Sleep consultant (0.55)                                    │
│                                                                 │
│ FASHION CROSS-CLUSTER:                                          │
│ → Athletic wear shopping                                        │
│ → Personal styling for new body                                 │
│                                                                 │
│ MILESTONE REWARDS:                                              │
│ → 5kg lost: Spa day discount                                   │
│ → 10kg lost: New wardrobe consultation                         │
│ → Goal reached: Professional photoshoot                        │
│                                                                 │
│ SUBSCRIPTION BUNDLES:                                           │
│ ┌────────────────────────────────────────────────────────────┐ │
│ │ "Fitness Starter" - ₦80,000/month                          │ │
│ │ Personal trainer (2x/week) + Meal plan                     │ │
│ ├────────────────────────────────────────────────────────────┤ │
│ │ "Total Transformation" - ₦200,000/month                    │ │
│ │ Trainer (4x/week) + Nutritionist + Meal prep + Yoga        │ │
│ ├────────────────────────────────────────────────────────────┤ │
│ │ "Premium Wellness" - ₦350,000/month                        │ │
│ │ All above + Weekly massage + Mental health support         │ │
│ └────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

### 8.4 User Journey: Pregnancy & Childbirth

```
TRIGGER: Pregnancy confirmed
┌─────────────────────────────────────────────────────────────────┐
│ TIMELINE: 9-month journey with predictable milestones          │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│ FIRST TRIMESTER (Weeks 1-12):                                   │
│ ├── OB-GYN recommendation (0.95)                               │
│ ├── Prenatal vitamins/pharmacy (0.90)                          │
│ ├── Healthy meal prep (0.75)                                   │
│ └── Pregnancy safe fitness (0.65)                              │
│                                                                 │
│ SECOND TRIMESTER (Weeks 13-26):                                 │
│ ├── Maternity wear (Fashion cross-cluster) (0.85)              │
│ ├── Prenatal yoga (0.80)                                       │
│ ├── Birth class/Educator (0.75)                                │
│ ├── Maternity photographer (0.70)                              │
│ └── Nursery planning (Home cross-cluster) (0.65)               │
│                                                                 │
│ THIRD TRIMESTER (Weeks 27-40):                                  │
│ ├── Doula services (0.75)                                      │
│ ├── Birth center/Hospital selection (0.85)                     │
│ ├── Baby supplies shopping (0.90)                              │
│ ├── Nursery setup (furniture, decor) (0.80)                    │
│ └── Baby shower planning (Celebrations cross-cluster) (0.70)   │
│                                                                 │
│ POST-DELIVERY (Weeks 1-12 post):                                │
│ ├── Post-natal nurse/care (0.85)                               │
│ ├── Lactation consultant (0.80)                                │
│ ├── Meal prep for new mom (0.78)                               │
│ ├── Newborn photographer (0.75)                                │
│ ├── House cleaning service (0.82)                              │
│ ├── Pediatrician recommendation (0.90)                         │
│ └── Postpartum fitness (0.65)                                  │
│                                                                 │
│ ONGOING (Months 3-12):                                          │
│ → Childcare/Nanny services                                      │
│ → Baby milestone photography                                    │
│ → Naming ceremony (Celebrations cross-cluster)                  │
│ → Pediatric services                                           │
│                                                                 │
│ MILESTONE BUNDLES:                                              │
│ "Expecting Mom" - ₦150K: Prenatal yoga + Nutrition + Maternity │
│ "Birth Ready" - ₦300K: Doula + Birth class + Hospital bag      │
│ "New Mom Care" - ₦250K: Post-natal nurse + Meals + Cleaning    │
└─────────────────────────────────────────────────────────────────┘
```

---

## Technical Architecture: Cross-Cluster Orchestration

### Multi-Cluster Event Detection

```
┌─────────────────────────────────────────────────────────────────┐
│              CROSS-CLUSTER ORCHESTRATION ENGINE                  │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                   EVENT DETECTOR                         │   │
│  │  Analyzes user behavior across all clusters              │   │
│  │  Identifies life events spanning multiple domains        │   │
│  └─────────────────────────────────────────────────────────┘   │
│                          │                                      │
│          ┌───────────────┼───────────────┐                     │
│          ▼               ▼               ▼                     │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐              │
│  │  WEDDING    │ │  RELOCATION │ │  NEW BABY   │              │
│  │  DETECTOR   │ │  DETECTOR   │ │  DETECTOR   │              │
│  └─────────────┘ └─────────────┘ └─────────────┘              │
│                                                                │
│  Cross-Cluster Signals:                                        │
│  ┌─────────────────────────────────────────────────────────┐  │
│  │ WEDDING (multi-cluster event):                          │  │
│  │ • Celebrations: Venue, catering, photography            │  │
│  │ • Fashion: Bridal styling, Aso-Ebi                      │  │
│  │ • Travel: Honeymoon, guest accommodation                │  │
│  │ • Home: Post-wedding home setup (if new couple)         │  │
│  └─────────────────────────────────────────────────────────┘  │
│                                                                │
│  ┌─────────────────────────────────────────────────────────┐  │
│  │ RELOCATION (multi-cluster event):                       │  │
│  │ • Home: Moving, cleaning, repairs, setup                │  │
│  │ • Travel: Transportation, temporary accommodation       │  │
│  │ • Business: Address changes, new service providers      │  │
│  │ • Education: School transfer (if family)                │  │
│  └─────────────────────────────────────────────────────────┘  │
│                                                                │
│  ┌─────────────────────────────────────────────────────────┐  │
│  │ NEW BABY (multi-cluster event):                         │  │
│  │ • Health: Prenatal, delivery, postnatal care            │  │
│  │ • Home: Nursery setup, childproofing                    │  │
│  │ • Celebrations: Baby shower, naming ceremony            │  │
│  │ • Fashion: Maternity wear, baby clothing                │  │
│  └─────────────────────────────────────────────────────────┘  │
│                                                                │
└─────────────────────────────────────────────────────────────────┘
```

### Unified User Context

```sql
-- Cross-cluster user context view
CREATE VIEW v_user_context AS
SELECT 
    u.id as user_id,
    u.life_stage,
    u.interests,
    
    -- Active projects across clusters
    (SELECT json_agg(json_build_object(
        'project_id', p.id,
        'event_type', p.event_type,
        'cluster', let.cluster_type,
        'status', p.status,
        'event_date', p.event_date
    ))
    FROM projects p
    JOIN life_event_triggers let ON let.id = p.event_trigger_id
    WHERE p.user_id = u.id AND p.status != 'completed') as active_projects,
    
    -- Recent bookings by cluster
    (SELECT json_object_agg(
        sc.cluster_type,
        booking_count
    )
    FROM (
        SELECT sc.cluster_type, COUNT(*) as booking_count
        FROM bookings b
        JOIN services s ON s.id = b.service_id
        JOIN service_categories sc ON sc.id = s.category_id
        WHERE b.user_id = u.id
        AND b.created_at > NOW() - INTERVAL '90 days'
        GROUP BY sc.cluster_type
    ) subq
    JOIN service_categories sc ON sc.cluster_type = subq.cluster_type) as recent_activity,
    
    -- Predicted needs
    (SELECT json_agg(json_build_object(
        'event_type', prediction.event_type,
        'confidence', prediction.confidence,
        'suggested_categories', prediction.categories
    ))
    FROM user_event_predictions prediction
    WHERE prediction.user_id = u.id
    AND prediction.confidence > 0.5) as predicted_needs
    
FROM users u;
```

---

*Continued in Part 3: Automotive, Creative, Property, Energy, Security, Pet clusters*
