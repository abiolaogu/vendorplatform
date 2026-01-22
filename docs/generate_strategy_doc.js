const {
  Document, Packer, Paragraph, TextRun, Table, TableRow, TableCell,
  HeadingLevel, AlignmentType, BorderStyle, WidthType, ShadingType,
  PageBreak, TableOfContents, Header, Footer, PageNumber
} = require('docx');
const fs = require('fs');

// Helper function to create styled paragraphs
const createParagraph = (text, options = {}) => {
  return new Paragraph({
    children: [new TextRun({ text, ...options.style })],
    ...options.paragraph
  });
};

// Create the document
const doc = new Document({
  styles: {
    default: {
      document: {
        run: { font: "Arial", size: 22 }
      }
    },
    paragraphStyles: [
      {
        id: "Heading1",
        name: "Heading 1",
        basedOn: "Normal",
        next: "Normal",
        quickFormat: true,
        run: { size: 36, bold: true, font: "Arial", color: "1a365d" },
        paragraph: { spacing: { before: 400, after: 200 }, outlineLevel: 0 }
      },
      {
        id: "Heading2",
        name: "Heading 2",
        basedOn: "Normal",
        next: "Normal",
        quickFormat: true,
        run: { size: 28, bold: true, font: "Arial", color: "2c5282" },
        paragraph: { spacing: { before: 300, after: 150 }, outlineLevel: 1 }
      },
      {
        id: "Heading3",
        name: "Heading 3",
        basedOn: "Normal",
        next: "Normal",
        quickFormat: true,
        run: { size: 24, bold: true, font: "Arial", color: "2b6cb0" },
        paragraph: { spacing: { before: 200, after: 100 }, outlineLevel: 2 }
      }
    ]
  },
  sections: [{
    properties: {
      page: {
        size: { width: 12240, height: 15840 },
        margin: { top: 1440, right: 1440, bottom: 1440, left: 1440 }
      }
    },
    headers: {
      default: new Header({
        children: [new Paragraph({
          children: [new TextRun({ text: "Vendor & Artisans Platform - Strategic Framework", italics: true, size: 18, color: "666666" })],
          alignment: AlignmentType.RIGHT
        })]
      })
    },
    footers: {
      default: new Footer({
        children: [new Paragraph({
          children: [
            new TextRun({ text: "Page ", size: 18 }),
            new TextRun({ children: [PageNumber.CURRENT], size: 18 }),
            new TextRun({ text: " of ", size: 18 }),
            new TextRun({ children: [PageNumber.TOTAL_PAGES], size: 18 })
          ],
          alignment: AlignmentType.CENTER
        })]
      })
    },
    children: [
      // TITLE PAGE
      new Paragraph({ spacing: { before: 2000 } }),
      new Paragraph({
        children: [new TextRun({ text: "VENDOR & ARTISANS PLATFORM", bold: true, size: 56, color: "1a365d" })],
        alignment: AlignmentType.CENTER
      }),
      new Paragraph({
        children: [new TextRun({ text: "Adjacent Opportunities Framework", size: 36, color: "2c5282" })],
        alignment: AlignmentType.CENTER,
        spacing: { before: 200 }
      }),
      new Paragraph({
        children: [new TextRun({ text: "Comprehensive Strategy, User Journeys, Technical Architecture & Business Models", size: 24, italics: true, color: "666666" })],
        alignment: AlignmentType.CENTER,
        spacing: { before: 400 }
      }),
      new Paragraph({
        children: [new TextRun({ text: "Version 1.0 | January 2026", size: 20 })],
        alignment: AlignmentType.CENTER,
        spacing: { before: 800 }
      }),
      new Paragraph({ children: [new PageBreak()] }),

      // TABLE OF CONTENTS
      new Paragraph({ heading: HeadingLevel.HEADING_1, children: [new TextRun("Table of Contents")] }),
      new TableOfContents("Table of Contents", { hyperlink: true, headingStyleRange: "1-3" }),
      new Paragraph({ children: [new PageBreak()] }),

      // EXECUTIVE SUMMARY
      new Paragraph({ heading: HeadingLevel.HEADING_1, children: [new TextRun("Executive Summary")] }),
      new Paragraph({
        children: [new TextRun({
          text: "This document presents a comprehensive strategic framework for building a vendor and artisans platform that leverages contextual commerce orchestration. The core insight is that when a customer needs one service, they almost always need 5-15 related services. By understanding and facilitating these adjacent service needs, the platform can capture significantly more value while providing superior customer experience."
        })]
      }),
      new Paragraph({ spacing: { before: 200 }, children: [new TextRun({ text: "Key Strategic Pillars:", bold: true })] }),
      new Paragraph({ children: [new TextRun("1. Life Event Detection: Automatically identify when users are experiencing major life events that trigger service cascades")] }),
      new Paragraph({ children: [new TextRun("2. Intelligent Adjacency Mapping: Maintain a dynamic graph of service relationships with context-aware scoring")] }),
      new Paragraph({ children: [new TextRun("3. Proactive Orchestration: Anticipate needs before users explicitly search, reducing friction and increasing conversion")] }),
      new Paragraph({ children: [new TextRun("4. Bundle Economics: Create value through coordinated multi-vendor packages with negotiated pricing")] }),
      new Paragraph({ children: [new TextRun("5. Network Effects: Build referral networks between vendors that strengthen platform stickiness")] }),
      new Paragraph({ children: [new PageBreak()] }),

      // SECTION 1: CLUSTER DEEP DIVES
      new Paragraph({ heading: HeadingLevel.HEADING_1, children: [new TextRun("Section 1: Service Cluster Deep Dives")] }),
      
      // CELEBRATIONS CLUSTER
      new Paragraph({ heading: HeadingLevel.HEADING_2, children: [new TextRun("1.1 Celebrations & Life Events Cluster")] }),
      new Paragraph({ children: [new TextRun({ text: "Market Overview:", bold: true })] }),
      new Paragraph({ children: [new TextRun("The celebrations market in Nigeria represents one of the highest-value service clusters, with Nigerians spending an estimated ₦2 trillion annually on weddings, funerals, and milestone celebrations. This cluster is characterized by high emotional investment, complex multi-vendor coordination needs, and significant price inelasticity for quality services.")] }),
      
      new Paragraph({ heading: HeadingLevel.HEADING_3, children: [new TextRun("1.1.1 Wedding Ecosystem - Complete User Journey")] }),
      new Paragraph({ children: [new TextRun({ text: "Phase 1: Engagement Announcement (Day 0-30)", bold: true, italics: true })] }),
      new Paragraph({ children: [new TextRun("Trigger Detection: User searches for 'engagement rings', 'proposal ideas', or changes relationship status")] }),
      new Paragraph({ children: [new TextRun("Immediate Adjacent Services: Engagement photographers, surprise planners, restaurant reservations")] }),
      new Paragraph({ children: [new TextRun("Platform Actions: Send congratulatory message, offer wedding planning starter kit, introduce project workspace")] }),
      
      new Paragraph({ spacing: { before: 200 }, children: [new TextRun({ text: "Phase 2: Planning Initiation (Day 30-180)", bold: true, italics: true })] }),
      new Paragraph({ children: [new TextRun("Primary Services: Event planners, venue scouts, budget calculators")] }),
      new Paragraph({ children: [new TextRun("Adjacent Cascade: Wedding venues → Catering → Decoration → Photography → Videography → Entertainment")] }),
      new Paragraph({ children: [new TextRun("Platform Actions: Create wedding project, suggest vendor shortlists based on budget/location/style, enable comparison tools")] }),
      
      new Paragraph({ spacing: { before: 200 }, children: [new TextRun({ text: "Phase 3: Vendor Selection (Day 60-120)", bold: true, italics: true })] }),
      new Paragraph({ children: [new TextRun("Critical Path Services: Venue (must book first), Caterer, Decorator, Photographer")] }),
      new Paragraph({ children: [new TextRun("Secondary Services: DJ/Entertainment, Cake baker, Florist, MC/Host")] }),
      new Paragraph({ children: [new TextRun("Support Services: Equipment rental, Lighting, Sound systems")] }),
      new Paragraph({ children: [new TextRun("Platform Actions: Coordinate vendor availability, negotiate bundle discounts, manage contracts")] }),
      
      new Paragraph({ spacing: { before: 200 }, children: [new TextRun({ text: "Phase 4: Personal Preparation (Day 90-7)", bold: true, italics: true })] }),
      new Paragraph({ children: [new TextRun("Bridal Services: Makeup artist, Hair stylist, Nail technician, Henna artist")] }),
      new Paragraph({ children: [new TextRun("Fashion Services: Tailors, Fabric vendors, Shoe makers, Jewelry")] }),
      new Paragraph({ children: [new TextRun("Adjacent Services: Fitness trainers, Skincare specialists, Spa services")] }),
      new Paragraph({ children: [new TextRun("Platform Actions: Schedule trial sessions, coordinate day-of timelines")] }),
      
      new Paragraph({ spacing: { before: 200 }, children: [new TextRun({ text: "Phase 5: Event Execution (Day 0)", bold: true, italics: true })] }),
      new Paragraph({ children: [new TextRun("Coordination Services: Event planner, Day-of coordinator")] }),
      new Paragraph({ children: [new TextRun("Support Services: Security, Ushers, Parking attendants, Cleanup crew")] }),
      new Paragraph({ children: [new TextRun("Platform Actions: Real-time vendor check-ins, issue resolution, payment releases")] }),
      
      new Paragraph({ spacing: { before: 200 }, children: [new TextRun({ text: "Phase 6: Post-Event (Day 1-30)", bold: true, italics: true })] }),
      new Paragraph({ children: [new TextRun("Follow-up Services: Photo/video editing, Album creation, Thank-you cards")] }),
      new Paragraph({ children: [new TextRun("New Trigger: Honeymoon → Travel cluster activation")] }),
      new Paragraph({ children: [new TextRun("Platform Actions: Collect reviews, process final payments, archive project")] }),
      
      new Paragraph({ spacing: { before: 200 }, children: [new TextRun({ text: "Typical Budget Allocation:", bold: true })] }),
      
      // Budget allocation table
      createBudgetTable([
        ['Category', 'Percentage', 'Typical Range (₦)'],
        ['Venue', '20-25%', '500,000 - 2,000,000'],
        ['Catering', '20-25%', '500,000 - 2,000,000'],
        ['Fashion & Beauty', '10-15%', '250,000 - 1,000,000'],
        ['Photography/Video', '8-12%', '200,000 - 800,000'],
        ['Decoration', '8-12%', '200,000 - 800,000'],
        ['Entertainment', '5-8%', '100,000 - 500,000'],
        ['Cake & Desserts', '3-5%', '50,000 - 300,000'],
        ['Stationery', '2-3%', '30,000 - 150,000'],
        ['Miscellaneous', '10-15%', '200,000 - 1,000,000']
      ]),
      
      new Paragraph({ children: [new PageBreak()] }),
      
      // FUNERAL ECOSYSTEM
      new Paragraph({ heading: HeadingLevel.HEADING_3, children: [new TextRun("1.1.2 Funeral/Memorial Services Ecosystem")] }),
      new Paragraph({ children: [new TextRun({ text: "Critical Insight:", bold: true }) , new TextRun(" This is an emotionally sensitive market where families need support during their most vulnerable moments. The platform must prioritize dignity, transparency, and compassionate service delivery.")] }),
      
      new Paragraph({ spacing: { before: 200 }, children: [new TextRun({ text: "User Journey:", bold: true })] }),
      new Paragraph({ children: [new TextRun("Trigger: Death notification (often sudden, always emotional)")] }),
      new Paragraph({ children: [new TextRun("Immediate Needs (0-24 hours): Mortuary services, death certificate processing, religious officiant notification")] }),
      new Paragraph({ children: [new TextRun("Planning Phase (1-7 days): Casket/urn selection, venue booking, catering arrangements, program printing")] }),
      new Paragraph({ children: [new TextRun("Ceremony Day: Transport coordination, tents/canopy, chairs, refreshments, photography (if culturally appropriate)")] }),
      new Paragraph({ children: [new TextRun("Post-Ceremony: Headstone engraving, memorial service planning, estate services referrals")] }),
      
      new Paragraph({ spacing: { before: 200 }, children: [new TextRun({ text: "Platform Value Proposition:", bold: true })] }),
      new Paragraph({ children: [new TextRun("Transparent, pre-negotiated pricing during a time when families are vulnerable to overcharging")] }),
      new Paragraph({ children: [new TextRun("Single point of coordination reduces stress on grieving families")] }),
      new Paragraph({ children: [new TextRun("Culturally-aware vendor matching (Christian, Muslim, Traditional, etc.)")] }),
      new Paragraph({ children: [new TextRun("Pre-need planning services for forward-thinking customers")] }),
      
      new Paragraph({ children: [new PageBreak()] }),
      
      // HOME CLUSTER
      new Paragraph({ heading: HeadingLevel.HEADING_2, children: [new TextRun("1.2 Home & Property Services Cluster")] }),
      new Paragraph({ children: [new TextRun("The home services cluster is characterized by high frequency, urgent needs, and strong geographic constraints. This cluster benefits from trust signals and repeat usage patterns.")] }),
      
      new Paragraph({ heading: HeadingLevel.HEADING_3, children: [new TextRun("1.2.1 Home Purchase & Relocation Ecosystem")] }),
      new Paragraph({ children: [new TextRun({ text: "Trigger Chain:", bold: true })] }),
      new Paragraph({ children: [new TextRun("Job change/Life event → Property search → Home purchase → Relocation → Home setup → Ongoing maintenance")] }),
      
      new Paragraph({ spacing: { before: 200 }, children: [new TextRun({ text: "Pre-Move Phase:", bold: true })] }),
      new Paragraph({ children: [new TextRun("Primary: Real estate agents, Property inspectors, Mortgage brokers")] }),
      new Paragraph({ children: [new TextRun("Adjacent: Interior designers, Architects (for renovations), Quantity surveyors")] }),
      
      new Paragraph({ spacing: { before: 200 }, children: [new TextRun({ text: "Moving Day Phase:", bold: true })] }),
      new Paragraph({ children: [new TextRun("Primary: Professional movers, Truck rental, Packing services")] }),
      new Paragraph({ children: [new TextRun("Adjacent: Cleaning (old place), Security during transport, Temporary storage")] }),
      
      new Paragraph({ spacing: { before: 200 }, children: [new TextRun({ text: "Setup Phase:", bold: true })] }),
      new Paragraph({ children: [new TextRun("Primary: Cleaning (new place), Furniture assembly, Appliance installation")] }),
      new Paragraph({ children: [new TextRun("Adjacent: Electricians, Plumbers, Painters, AC technicians, Pest control")] }),
      new Paragraph({ children: [new TextRun("Security: CCTV installation, Smart locks, Security system setup")] }),
      
      new Paragraph({ spacing: { before: 200 }, children: [new TextRun({ text: "Settling Phase:", bold: true })] }),
      new Paragraph({ children: [new TextRun("Interior: Curtains/blinds, Interior decoration, Landscaping")] }),
      new Paragraph({ children: [new TextRun("Utilities: Internet setup, Generator/solar installation, Water treatment")] }),
      new Paragraph({ children: [new TextRun("Admin: Address change services, Utility connections, Local orientation")] }),
      
      new Paragraph({ heading: HeadingLevel.HEADING_3, children: [new TextRun("1.2.2 Home Renovation Ecosystem")] }),
      new Paragraph({ children: [new TextRun({ text: "Renovation Triggers:", bold: true })] }),
      new Paragraph({ children: [new TextRun("Major Life Events: New home, Growing family, Empty nest, Work-from-home transition")] }),
      new Paragraph({ children: [new TextRun("Property Events: Damage repair, Value improvement, Style update")] }),
      new Paragraph({ children: [new TextRun("Planned Maintenance: Kitchen upgrade, Bathroom remodel, Outdoor improvement")] }),
      
      new Paragraph({ spacing: { before: 200 }, children: [new TextRun({ text: "Service Dependencies (Critical Path):", bold: true })] }),
      new Paragraph({ children: [new TextRun("1. Design Phase: Architect → Interior Designer → Quantity Surveyor")] }),
      new Paragraph({ children: [new TextRun("2. Structural: Contractor → Structural Engineer → Building Inspector")] }),
      new Paragraph({ children: [new TextRun("3. Systems: Electrician → Plumber → HVAC Technician")] }),
      new Paragraph({ children: [new TextRun("4. Finishing: Tiler → Painter → Carpenter → Decorator")] }),
      
      new Paragraph({ children: [new PageBreak()] }),
      
      // TRAVEL CLUSTER
      new Paragraph({ heading: HeadingLevel.HEADING_2, children: [new TextRun("1.3 Travel & Mobility Cluster")] }),
      new Paragraph({ children: [new TextRun("Travel represents a unique cluster where service needs are geographically distributed across origin and destination locations, requiring multi-city vendor networks.")] }),
      
      new Paragraph({ heading: HeadingLevel.HEADING_3, children: [new TextRun("1.3.1 Domestic Flight Travel Ecosystem")] }),
      new Paragraph({ children: [new TextRun({ text: "Service Timeline:", bold: true })] }),
      new Paragraph({ children: [new TextRun("Pre-Departure (Origin City):")] }),
      new Paragraph({ children: [new TextRun("  - Taxi to airport / Airport parking")] }),
      new Paragraph({ children: [new TextRun("  - Porter services / Fast-track check-in")] }),
      new Paragraph({ children: [new TextRun("  - Airport lounge access")] }),
      new Paragraph({ children: [new TextRun("Arrival (Destination City):")] }),
      new Paragraph({ children: [new TextRun("  - Airport pickup / Taxi service")] }),
      new Paragraph({ children: [new TextRun("  - Car rental (if needed)")] }),
      new Paragraph({ children: [new TextRun("  - Hotel / Accommodation")] }),
      new Paragraph({ children: [new TextRun("During Stay:")] }),
      new Paragraph({ children: [new TextRun("  - Chauffeur services / Daily transport")] }),
      new Paragraph({ children: [new TextRun("  - Restaurant reservations")] }),
      new Paragraph({ children: [new TextRun("  - Local experiences / Tours")] }),
      
      new Paragraph({ heading: HeadingLevel.HEADING_3, children: [new TextRun("1.3.2 International Travel Ecosystem")] }),
      new Paragraph({ children: [new TextRun({ text: "Extended Service Chain:", bold: true })] }),
      new Paragraph({ children: [new TextRun("Documentation Phase (30-90 days before):")] }),
      new Paragraph({ children: [new TextRun("  - Visa consultants / Immigration advisors")] }),
      new Paragraph({ children: [new TextRun("  - Document attestation services")] }),
      new Paragraph({ children: [new TextRun("  - Passport renewal (if needed)")] }),
      new Paragraph({ children: [new TextRun("  - Travel insurance")] }),
      new Paragraph({ children: [new TextRun("Preparation Phase (7-30 days before):")] }),
      new Paragraph({ children: [new TextRun("  - Currency exchange / Forex cards")] }),
      new Paragraph({ children: [new TextRun("  - Luggage purchase / Travel gear")] }),
      new Paragraph({ children: [new TextRun("  - International data plans / SIM cards")] }),
      new Paragraph({ children: [new TextRun("  - Vaccinations / Medical preparations")] }),
      new Paragraph({ children: [new TextRun("Travel Day Services: Origin + Destination airport services")] }),
      new Paragraph({ children: [new TextRun("Return: Reverse logistics + Settling back services")] }),
      
      new Paragraph({ children: [new PageBreak()] }),
      
      // HORECA CLUSTER
      new Paragraph({ heading: HeadingLevel.HEADING_2, children: [new TextRun("1.4 Food & Hospitality (HORECA) Cluster")] }),
      new Paragraph({ children: [new TextRun("The HORECA cluster spans both B2C (private dining, meal prep) and B2B (restaurant supply, food business launch) opportunities.")] }),
      
      new Paragraph({ heading: HeadingLevel.HEADING_3, children: [new TextRun("1.4.1 Restaurant Launch Ecosystem")] }),
      new Paragraph({ children: [new TextRun({ text: "Complete Launch Timeline:", bold: true })] }),
      new Paragraph({ children: [new TextRun("Phase 1 - Concept & Planning (Month 1-2):")] }),
      new Paragraph({ children: [new TextRun("  - Restaurant consultants → Business plan development")] }),
      new Paragraph({ children: [new TextRun("  - Location scouts → Site analysis")] }),
      new Paragraph({ children: [new TextRun("  - Menu developers → Concept refinement")] }),
      new Paragraph({ children: [new TextRun("Phase 2 - Setup (Month 2-4):")] }),
      new Paragraph({ children: [new TextRun("  - Interior designers → Space planning")] }),
      new Paragraph({ children: [new TextRun("  - Kitchen equipment vendors → Equipment sourcing")] }),
      new Paragraph({ children: [new TextRun("  - Contractors → Build-out")] }),
      new Paragraph({ children: [new TextRun("  - POS system vendors → Technology setup")] }),
      new Paragraph({ children: [new TextRun("Phase 3 - Licensing (Month 3-5):")] }),
      new Paragraph({ children: [new TextRun("  - Food safety consultants → NAFDAC compliance")] }),
      new Paragraph({ children: [new TextRun("  - Health inspectors → Certification")] }),
      new Paragraph({ children: [new TextRun("  - Business registration → Legal setup")] }),
      new Paragraph({ children: [new TextRun("Phase 4 - Staffing (Month 4-5):")] }),
      new Paragraph({ children: [new TextRun("  - HR/Recruitment → Team hiring")] }),
      new Paragraph({ children: [new TextRun("  - Training providers → Staff development")] }),
      new Paragraph({ children: [new TextRun("  - Uniform suppliers → Staff outfitting")] }),
      new Paragraph({ children: [new TextRun("Phase 5 - Marketing Launch (Month 5-6):")] }),
      new Paragraph({ children: [new TextRun("  - Food photographers → Menu photography")] }),
      new Paragraph({ children: [new TextRun("  - Menu designers → Print materials")] }),
      new Paragraph({ children: [new TextRun("  - Social media managers → Online presence")] }),
      new Paragraph({ children: [new TextRun("  - Influencer coordinators → Launch buzz")] }),
      new Paragraph({ children: [new TextRun("Phase 6 - Operations (Ongoing):")] }),
      new Paragraph({ children: [new TextRun("  - Food suppliers → Ingredient sourcing")] }),
      new Paragraph({ children: [new TextRun("  - Cleaning services → Hygiene maintenance")] }),
      new Paragraph({ children: [new TextRun("  - Equipment maintenance → Operational continuity")] }),
      
      new Paragraph({ children: [new PageBreak()] }),
      
      // BUSINESS CLUSTER
      new Paragraph({ heading: HeadingLevel.HEADING_2, children: [new TextRun("1.5 Business Services Cluster")] }),
      
      new Paragraph({ heading: HeadingLevel.HEADING_3, children: [new TextRun("1.5.1 Business Launch Ecosystem")] }),
      new Paragraph({ children: [new TextRun({ text: "Startup Service Chain:", bold: true })] }),
      new Paragraph({ children: [new TextRun("Legal Foundation:")] }),
      new Paragraph({ children: [new TextRun("  Business Registration → Legal Services → Accounting Setup → Tax Registration")] }),
      new Paragraph({ children: [new TextRun("Physical Setup:")] }),
      new Paragraph({ children: [new TextRun("  Office Space → Furniture → IT Infrastructure → Security Systems")] }),
      new Paragraph({ children: [new TextRun("Brand Building:")] }),
      new Paragraph({ children: [new TextRun("  Branding/Design → Website Development → Marketing → PR")] }),
      new Paragraph({ children: [new TextRun("Operations:")] }),
      new Paragraph({ children: [new TextRun("  HR/Recruitment → Payroll Setup → Insurance → Banking")] }),
      
      new Paragraph({ heading: HeadingLevel.HEADING_3, children: [new TextRun("1.5.2 Corporate Event Ecosystem")] }),
      new Paragraph({ children: [new TextRun({ text: "Event Types & Service Needs:", bold: true })] }),
      new Paragraph({ children: [new TextRun("Conferences:")] }),
      new Paragraph({ children: [new TextRun("  Venue + AV Equipment + Catering + Registration + Badges + Photography + Transcription")] }),
      new Paragraph({ children: [new TextRun("Product Launches:")] }),
      new Paragraph({ children: [new TextRun("  Event Planners + PR + Media + Influencers + Demo Technicians + Photographers")] }),
      new Paragraph({ children: [new TextRun("Team Building:")] }),
      new Paragraph({ children: [new TextRun("  Activity Organizers + Facilitators + Transport + Outdoor Equipment + Catering")] }),
      new Paragraph({ children: [new TextRun("AGMs:")] }),
      new Paragraph({ children: [new TextRun("  Venue + Legal Support + Document Printing + Security + Refreshments")] }),
      
      new Paragraph({ children: [new PageBreak()] }),
      
      // HEALTH CLUSTER
      new Paragraph({ heading: HeadingLevel.HEADING_2, children: [new TextRun("1.6 Health & Wellness Cluster")] }),
      
      new Paragraph({ heading: HeadingLevel.HEADING_3, children: [new TextRun("1.6.1 Medical Recovery Ecosystem")] }),
      new Paragraph({ children: [new TextRun({ text: "Service Progression:", bold: true })] }),
      new Paragraph({ children: [new TextRun("Pre-Procedure: Second opinions, Diagnostic tests, Insurance verification")] }),
      new Paragraph({ children: [new TextRun("Hospital Phase: Room booking, Specialist coordination, Family accommodation")] }),
      new Paragraph({ children: [new TextRun("Recovery Phase:")] }),
      new Paragraph({ children: [new TextRun("  - Home nursing care")] }),
      new Paragraph({ children: [new TextRun("  - Physiotherapy")] }),
      new Paragraph({ children: [new TextRun("  - Medical equipment rental")] }),
      new Paragraph({ children: [new TextRun("  - Specialized meal preparation")] }),
      new Paragraph({ children: [new TextRun("  - Home modifications (if needed)")] }),
      new Paragraph({ children: [new TextRun("Follow-up: Telemedicine, Lab tests at home, Medication delivery")] }),
      
      new Paragraph({ heading: HeadingLevel.HEADING_3, children: [new TextRun("1.6.2 Fitness Transformation Ecosystem")] }),
      new Paragraph({ children: [new TextRun({ text: "Holistic Transformation Services:", bold: true })] }),
      new Paragraph({ children: [new TextRun("Training: Personal trainers, Gym memberships, Home equipment")] }),
      new Paragraph({ children: [new TextRun("Nutrition: Nutritionists, Meal prep services, Supplement vendors")] }),
      new Paragraph({ children: [new TextRun("Mind-Body: Yoga instructors, Meditation guides, Mental health support")] }),
      new Paragraph({ children: [new TextRun("Recovery: Spa services, Massage therapists, Physiotherapists")] }),
      new Paragraph({ children: [new TextRun("Tracking: Health apps, Wearable devices, Lab testing")] }),
      
      new Paragraph({ children: [new PageBreak()] }),
      
      // SECTION 2: TECHNICAL ARCHITECTURE
      new Paragraph({ heading: HeadingLevel.HEADING_1, children: [new TextRun("Section 2: Technical Architecture")] }),
      
      new Paragraph({ heading: HeadingLevel.HEADING_2, children: [new TextRun("2.1 System Architecture Overview")] }),
      new Paragraph({ children: [new TextRun("The platform is built on a microservices architecture optimized for scalability, real-time recommendations, and multi-tenant vendor management.")] }),
      
      new Paragraph({ spacing: { before: 200 }, children: [new TextRun({ text: "Core Components:", bold: true })] }),
      new Paragraph({ children: [new TextRun("1. API Gateway (Go): High-performance request routing, rate limiting, authentication")] }),
      new Paragraph({ children: [new TextRun("2. Recommendation Engine (Go + Python): Real-time adjacency calculations, ML-powered personalization")] }),
      new Paragraph({ children: [new TextRun("3. Event Detection Service (Python): NLP-based life event identification from user signals")] }),
      new Paragraph({ children: [new TextRun("4. Booking Orchestrator (Go): Transaction management, vendor coordination, payment processing")] }),
      new Paragraph({ children: [new TextRun("5. Search Service (Elasticsearch): Full-text search, faceted filtering, geo-queries")] }),
      new Paragraph({ children: [new TextRun("6. Analytics Pipeline (Python + ClickHouse): Real-time metrics, cohort analysis, A/B testing")] }),
      
      new Paragraph({ heading: HeadingLevel.HEADING_2, children: [new TextRun("2.2 Database Architecture")] }),
      new Paragraph({ children: [new TextRun({ text: "Primary Database: PostgreSQL 15+", bold: true })] }),
      new Paragraph({ children: [new TextRun("Extensions: PostGIS (geospatial), TimescaleDB (time-series), pg_trgm (fuzzy search)")] }),
      new Paragraph({ children: [new TextRun("Key Tables:")] }),
      new Paragraph({ children: [new TextRun("  - users: Customer profiles, preferences, lifetime value")] }),
      new Paragraph({ children: [new TextRun("  - vendors: Vendor profiles, verification status, service areas")] }),
      new Paragraph({ children: [new TextRun("  - services: Individual service offerings with pricing")] }),
      new Paragraph({ children: [new TextRun("  - service_categories: Hierarchical category taxonomy (4 levels)")] }),
      new Paragraph({ children: [new TextRun("  - service_adjacencies: Core adjacency graph with context-aware scores")] }),
      new Paragraph({ children: [new TextRun("  - life_event_triggers: Event definitions and detection patterns")] }),
      new Paragraph({ children: [new TextRun("  - projects: User event projects grouping related bookings")] }),
      new Paragraph({ children: [new TextRun("  - bookings: Transaction records with attribution tracking")] }),
      
      new Paragraph({ spacing: { before: 200 }, children: [new TextRun({ text: "Caching Layer: Redis", bold: true })] }),
      new Paragraph({ children: [new TextRun("Use Cases: Session storage, adjacency graph cache, trending services, rate limiting")] }),
      
      new Paragraph({ heading: HeadingLevel.HEADING_2, children: [new TextRun("2.3 Recommendation Engine Architecture")] }),
      new Paragraph({ children: [new TextRun({ text: "Multi-Strategy Approach:", bold: true })] }),
      new Paragraph({ children: [new TextRun("1. Adjacency-Based (35% weight): Pre-computed category relationships with context-aware scoring")] }),
      new Paragraph({ children: [new TextRun("2. Collaborative Filtering (25% weight): User-user similarity based on booking patterns")] }),
      new Paragraph({ children: [new TextRun("3. Trending (15% weight): Real-time popularity signals with recency decay")] }),
      new Paragraph({ children: [new TextRun("4. Personalization (20% weight): User preference matching, life stage alignment")] }),
      new Paragraph({ children: [new TextRun("5. Location (5% weight): Geographic relevance and vendor coverage")] }),
      
      new Paragraph({ spacing: { before: 200 }, children: [new TextRun({ text: "Diversification:", bold: true })] }),
      new Paragraph({ children: [new TextRun("Algorithm: Maximal Marginal Relevance (MMR)")] }),
      new Paragraph({ children: [new TextRun("Purpose: Ensure category diversity, prevent vendor monopolization in results")] }),
      new Paragraph({ children: [new TextRun("Tunable: Diversity factor (0-1) adjustable per request")] }),
      
      new Paragraph({ children: [new PageBreak()] }),
      
      // SECTION 3: BUSINESS MODEL CANVASES
      new Paragraph({ heading: HeadingLevel.HEADING_1, children: [new TextRun("Section 3: Business Model Canvases")] }),
      
      new Paragraph({ heading: HeadingLevel.HEADING_2, children: [new TextRun("3.1 LifeOS - The Super-App")] }),
      createBusinessModelCanvas({
        name: "LifeOS - Life Event Orchestration Platform",
        valueProposition: "AI-powered life orchestration that detects major life events and automatically coordinates all needed services through a single platform, reducing planning friction by 80%",
        customerSegments: [
          "Primary: Urban professionals (25-45) experiencing major life events",
          "Secondary: Busy parents managing family milestones",
          "Tertiary: Event planners seeking coordination tools"
        ],
        channels: [
          "Mobile app (primary)",
          "Web platform",
          "WhatsApp integration",
          "Partner referrals"
        ],
        customerRelationships: [
          "AI concierge (automated)",
          "Human coordinators (premium)",
          "Community (reviews, recommendations)"
        ],
        revenueStreams: [
          "Transaction fees (8-15% per booking)",
          "Premium subscriptions (₦5,000-20,000/month)",
          "Bundle commissions (additional 2-5%)",
          "Vendor subscriptions (₦10,000-100,000/month)",
          "Lead generation fees",
          "Event financing (interest revenue)"
        ],
        keyResources: [
          "Adjacency graph database",
          "ML recommendation models",
          "Verified vendor network",
          "User behavior data",
          "Brand trust"
        ],
        keyActivities: [
          "Life event detection algorithm development",
          "Vendor acquisition and verification",
          "Platform development and maintenance",
          "Quality assurance and dispute resolution",
          "Data analysis and personalization"
        ],
        keyPartnerships: [
          "Payment processors (Paystack, Flutterwave)",
          "Insurance providers",
          "Banks (financing)",
          "Telcos (USSD, SMS)",
          "Social media platforms (detection signals)"
        ],
        costStructure: [
          "Technology infrastructure (30%)",
          "Customer acquisition (25%)",
          "Vendor acquisition (15%)",
          "Operations & support (20%)",
          "G&A (10%)"
        ]
      }),
      
      new Paragraph({ children: [new PageBreak()] }),
      
      new Paragraph({ heading: HeadingLevel.HEADING_2, children: [new TextRun("3.2 EventGPT - AI Event Planner")] }),
      createBusinessModelCanvas({
        name: "EventGPT - Conversational Event Planning",
        valueProposition: "Natural language event planning: describe your event in plain language and get an AI-generated vendor plan, complete budget, and one-click booking",
        customerSegments: [
          "First-time event planners (weddings, parties)",
          "Busy professionals with no time to plan",
          "Budget-conscious planners seeking optimization",
          "Couples planning weddings remotely"
        ],
        channels: [
          "Chat interface (web/mobile)",
          "WhatsApp bot",
          "Voice assistant integration",
          "Social media (Instagram DM)"
        ],
        customerRelationships: [
          "Conversational AI (primary)",
          "Human escalation (complex events)",
          "Proactive check-ins and reminders"
        ],
        revenueStreams: [
          "Success fees (% of event budget)",
          "Premium AI features subscription",
          "Vendor placement fees",
          "Planning templates marketplace",
          "White-label licensing to venues"
        ],
        keyResources: [
          "Large language model (fine-tuned)",
          "Event knowledge base",
          "Vendor inventory system",
          "Pricing intelligence",
          "User conversation data"
        ],
        keyActivities: [
          "AI model training and refinement",
          "Vendor data enrichment",
          "Conversation design",
          "Integration development",
          "User research"
        ],
        keyPartnerships: [
          "AI/LLM providers (OpenAI, Anthropic)",
          "Venue partners (exclusive deals)",
          "Wedding/event blogs (content)",
          "Financial institutions (event loans)"
        ],
        costStructure: [
          "AI/ML infrastructure (35%)",
          "Engineering (30%)",
          "Marketing (20%)",
          "Operations (15%)"
        ]
      }),
      
      new Paragraph({ children: [new PageBreak()] }),
      
      new Paragraph({ heading: HeadingLevel.HEADING_2, children: [new TextRun("3.3 VendorNet - B2B Marketplace")] }),
      createBusinessModelCanvas({
        name: "VendorNet - Vendor Partnership Network",
        valueProposition: "Vendor-to-vendor referral network where service providers earn commissions by referring clients to complementary vendors, creating a self-reinforcing ecosystem",
        customerSegments: [
          "Primary: Active vendors on platform",
          "Secondary: Vendor partnerships/collectives",
          "Tertiary: Event venues seeking vendor networks"
        ],
        channels: [
          "Vendor dashboard (web)",
          "Mobile app for vendors",
          "API for enterprise integration",
          "WhatsApp for quick referrals"
        ],
        customerRelationships: [
          "Automated referral matching",
          "Performance analytics",
          "Partnership facilitation",
          "Dispute mediation"
        ],
        revenueStreams: [
          "Referral processing fees (2-5%)",
          "Premium vendor subscriptions",
          "Featured placement",
          "API access fees",
          "Data insights packages"
        ],
        keyResources: [
          "Vendor relationship graph",
          "Referral attribution system",
          "Payment splitting infrastructure",
          "Trust scoring algorithm"
        ],
        keyActivities: [
          "Vendor onboarding",
          "Referral tracking",
          "Payment processing",
          "Network analysis",
          "Quality enforcement"
        ],
        keyPartnerships: [
          "Industry associations",
          "Training providers",
          "Equipment suppliers",
          "Insurance companies"
        ],
        costStructure: [
          "Payment processing (15%)",
          "Technology (35%)",
          "Vendor success (25%)",
          "Marketing (15%)",
          "Operations (10%)"
        ]
      }),
      
      new Paragraph({ children: [new PageBreak()] }),
      
      new Paragraph({ heading: HeadingLevel.HEADING_2, children: [new TextRun("3.4 HomeRescue - Emergency Services")] }),
      createBusinessModelCanvas({
        name: "HomeRescue - Home Emergency Response",
        valueProposition: "One-button home emergency response with guaranteed SLA. When pipes burst or power fails, get verified help within 60 minutes with transparent, pre-negotiated pricing",
        customerSegments: [
          "Homeowners (primary)",
          "Property managers",
          "Landlords",
          "Estate management companies"
        ],
        channels: [
          "Emergency hotline",
          "Mobile app (one-tap)",
          "Smart home integration",
          "Property management portals"
        ],
        customerRelationships: [
          "24/7 dispatch center",
          "Automated status updates",
          "Post-service follow-up",
          "Preventive maintenance reminders"
        ],
        revenueStreams: [
          "Emergency service fees (premium pricing)",
          "Subscription plans (guaranteed response)",
          "Insurance claim processing fees",
          "Preventive maintenance contracts",
          "Equipment sales/rental"
        ],
        keyResources: [
          "Verified emergency vendor network",
          "Dispatch/routing system",
          "24/7 operations center",
          "SLA monitoring infrastructure"
        ],
        keyActivities: [
          "Vendor vetting (speed + quality)",
          "Real-time dispatch optimization",
          "SLA monitoring and enforcement",
          "Insurance partner coordination",
          "Quality assurance"
        ],
        keyPartnerships: [
          "Insurance companies",
          "Smart home providers",
          "Property developers",
          "Estate associations",
          "Equipment suppliers"
        ],
        costStructure: [
          "Operations center (30%)",
          "Technology (25%)",
          "Vendor network management (20%)",
          "Marketing (15%)",
          "Insurance/legal (10%)"
        ]
      }),
      
      new Paragraph({ children: [new PageBreak()] }),
      
      // SECTION 4: IMPLEMENTATION ROADMAP
      new Paragraph({ heading: HeadingLevel.HEADING_1, children: [new TextRun("Section 4: Implementation Roadmap")] }),
      
      new Paragraph({ heading: HeadingLevel.HEADING_2, children: [new TextRun("4.1 Phase 1: Foundation (Months 1-3)")] }),
      new Paragraph({ children: [new TextRun({ text: "Objectives:", bold: true })] }),
      new Paragraph({ children: [new TextRun("- Deploy core database schema with adjacency graph")] }),
      new Paragraph({ children: [new TextRun("- Launch basic vendor onboarding for 2-3 clusters")] }),
      new Paragraph({ children: [new TextRun("- Implement adjacency-based recommendations (v1)")] }),
      new Paragraph({ children: [new TextRun("- Build MVP booking flow")] }),
      new Paragraph({ children: [new TextRun({ text: "Target Clusters:", bold: true }) , new TextRun(" Celebrations (weddings), Home Services")] }),
      new Paragraph({ children: [new TextRun({ text: "Success Metrics:", bold: true }) , new TextRun(" 100 vendors, 500 users, 50 bookings, 20% cross-sell rate")] }),
      
      new Paragraph({ heading: HeadingLevel.HEADING_2, children: [new TextRun("4.2 Phase 2: Intelligence (Months 4-6)")] }),
      new Paragraph({ children: [new TextRun({ text: "Objectives:", bold: true })] }),
      new Paragraph({ children: [new TextRun("- Implement ML-based event detection")] }),
      new Paragraph({ children: [new TextRun("- Launch project workspace for multi-booking coordination")] }),
      new Paragraph({ children: [new TextRun("- Add collaborative filtering to recommendations")] }),
      new Paragraph({ children: [new TextRun("- Introduce bundle pricing engine")] }),
      new Paragraph({ children: [new TextRun({ text: "New Clusters:", bold: true }) , new TextRun(" Travel, HORECA")] }),
      new Paragraph({ children: [new TextRun({ text: "Success Metrics:", bold: true }) , new TextRun(" 500 vendors, 5,000 users, 500 bookings, 35% cross-sell rate")] }),
      
      new Paragraph({ heading: HeadingLevel.HEADING_2, children: [new TextRun("4.3 Phase 3: Scale (Months 7-12)")] }),
      new Paragraph({ children: [new TextRun({ text: "Objectives:", bold: true })] }),
      new Paragraph({ children: [new TextRun("- Launch vendor partnership network")] }),
      new Paragraph({ children: [new TextRun("- Implement real-time demand forecasting")] }),
      new Paragraph({ children: [new TextRun("- Add financing options for large events")] }),
      new Paragraph({ children: [new TextRun("- Geographic expansion (3+ cities)")] }),
      new Paragraph({ children: [new TextRun({ text: "All Clusters:", bold: true }) , new TextRun(" Business, Health, Automotive, Property, Energy, Creative, Security, Pet")] }),
      new Paragraph({ children: [new TextRun({ text: "Success Metrics:", bold: true }) , new TextRun(" 5,000 vendors, 50,000 users, 10,000 bookings, 50% cross-sell rate, ₦500M GMV")] }),
      
      new Paragraph({ children: [new PageBreak()] }),
      
      // SECTION 5: KEY METRICS
      new Paragraph({ heading: HeadingLevel.HEADING_1, children: [new TextRun("Section 5: Key Performance Indicators")] }),
      
      new Paragraph({ heading: HeadingLevel.HEADING_2, children: [new TextRun("5.1 Platform Health Metrics")] }),
      
      createMetricsTable([
        ['Metric', 'Definition', 'Target'],
        ['Cross-Sell Rate', 'Bookings from recommendations / Total bookings', '> 40%'],
        ['Services per Project', 'Average services booked per user project', '> 5'],
        ['Recommendation CTR', 'Clicks on recommendations / Impressions', '> 15%'],
        ['Recommendation Conversion', 'Bookings from clicks / Clicks', '> 20%'],
        ['Vendor Network Density', 'Avg partnerships per vendor', '> 8'],
        ['Event Detection Accuracy', 'Correctly identified events / Detected events', '> 80%'],
        ['Bundle Attach Rate', 'Bundles sold / Bundle-eligible bookings', '> 25%'],
        ['NPS (Customers)', 'Net Promoter Score', '> 50'],
        ['NPS (Vendors)', 'Net Promoter Score', '> 40']
      ]),
      
      new Paragraph({ heading: HeadingLevel.HEADING_2, children: [new TextRun("5.2 Financial Metrics")] }),
      
      createMetricsTable([
        ['Metric', 'Definition', 'Target (Year 1)'],
        ['GMV', 'Gross Merchandise Value', '₦1B'],
        ['Net Revenue', 'Total platform revenue', '₦100M'],
        ['Take Rate', 'Revenue / GMV', '10-12%'],
        ['CAC', 'Customer Acquisition Cost', '< ₦2,000'],
        ['LTV', 'Customer Lifetime Value', '> ₦20,000'],
        ['LTV:CAC Ratio', 'Lifetime Value / Acquisition Cost', '> 10:1'],
        ['Vendor ARPU', 'Average Revenue Per Vendor', '₦50,000/month'],
        ['Contribution Margin', 'Gross Profit / Revenue', '> 60%']
      ]),
      
      new Paragraph({ children: [new PageBreak()] }),
      
      // CONCLUSION
      new Paragraph({ heading: HeadingLevel.HEADING_1, children: [new TextRun("Conclusion")] }),
      new Paragraph({ children: [new TextRun("The Vendor & Artisans Platform represents a significant opportunity to transform how Nigerians access and coordinate services during major life events. By leveraging contextual commerce orchestration, the platform can:")] }),
      new Paragraph({ spacing: { before: 200 }, children: [new TextRun("1. Capture 5-15x more value per customer by facilitating adjacent service needs")] }),
      new Paragraph({ children: [new TextRun("2. Reduce customer friction through proactive recommendations and coordinated bookings")] }),
      new Paragraph({ children: [new TextRun("3. Create vendor lock-in through partnership networks and referral economics")] }),
      new Paragraph({ children: [new TextRun("4. Build defensible data assets through behavioral understanding and adjacency intelligence")] }),
      new Paragraph({ children: [new TextRun("5. Enable premium pricing through quality assurance and convenience")] }),
      new Paragraph({ spacing: { before: 200 }, children: [new TextRun("The technical foundation—combining high-performance Go services with Python ML capabilities and a carefully designed PostgreSQL schema—provides the scalability and intelligence needed to execute this vision.")] }),
      new Paragraph({ spacing: { before: 200 }, children: [new TextRun("Success will be measured not just by transaction volume, but by the platform's ability to become the default coordination layer for Nigerian life events—the operating system for celebrations, transitions, and daily services.")] }),
    ]
  }]
});

// Helper function to create budget tables
function createBudgetTable(rows) {
  const border = { style: BorderStyle.SINGLE, size: 1, color: "CCCCCC" };
  const borders = { top: border, bottom: border, left: border, right: border };
  
  return new Table({
    width: { size: 100, type: WidthType.PERCENTAGE },
    columnWidths: [3500, 2000, 3500],
    rows: rows.map((row, index) => new TableRow({
      children: row.map(cell => new TableCell({
        borders,
        shading: { 
          fill: index === 0 ? "2c5282" : (index % 2 === 0 ? "f7fafc" : "ffffff"),
          type: ShadingType.CLEAR 
        },
        margins: { top: 80, bottom: 80, left: 120, right: 120 },
        children: [new Paragraph({
          children: [new TextRun({ 
            text: cell, 
            bold: index === 0,
            color: index === 0 ? "ffffff" : "000000",
            size: 20
          })]
        })]
      }))
    }))
  });
}

// Helper function to create metrics tables
function createMetricsTable(rows) {
  const border = { style: BorderStyle.SINGLE, size: 1, color: "CCCCCC" };
  const borders = { top: border, bottom: border, left: border, right: border };
  
  return new Table({
    width: { size: 100, type: WidthType.PERCENTAGE },
    columnWidths: [3000, 4000, 2000],
    rows: rows.map((row, index) => new TableRow({
      children: row.map(cell => new TableCell({
        borders,
        shading: { 
          fill: index === 0 ? "1a365d" : (index % 2 === 0 ? "f7fafc" : "ffffff"),
          type: ShadingType.CLEAR 
        },
        margins: { top: 80, bottom: 80, left: 120, right: 120 },
        children: [new Paragraph({
          children: [new TextRun({ 
            text: cell, 
            bold: index === 0,
            color: index === 0 ? "ffffff" : "000000",
            size: 20
          })]
        })]
      }))
    }))
  });
}

// Helper function to create business model canvas
function createBusinessModelCanvas(model) {
  const sections = [
    new Paragraph({ spacing: { before: 200 }, children: [new TextRun({ text: `Platform: ${model.name}`, bold: true, size: 24 })] }),
    new Paragraph({ spacing: { before: 100 }, children: [new TextRun({ text: "Value Proposition: ", bold: true }), new TextRun(model.valueProposition)] }),
    new Paragraph({ spacing: { before: 150 }, children: [new TextRun({ text: "Customer Segments:", bold: true })] }),
    ...model.customerSegments.map(s => new Paragraph({ children: [new TextRun(`  • ${s}`)] })),
    new Paragraph({ spacing: { before: 150 }, children: [new TextRun({ text: "Channels:", bold: true })] }),
    ...model.channels.map(c => new Paragraph({ children: [new TextRun(`  • ${c}`)] })),
    new Paragraph({ spacing: { before: 150 }, children: [new TextRun({ text: "Revenue Streams:", bold: true })] }),
    ...model.revenueStreams.map(r => new Paragraph({ children: [new TextRun(`  • ${r}`)] })),
    new Paragraph({ spacing: { before: 150 }, children: [new TextRun({ text: "Key Resources:", bold: true })] }),
    ...model.keyResources.map(r => new Paragraph({ children: [new TextRun(`  • ${r}`)] })),
    new Paragraph({ spacing: { before: 150 }, children: [new TextRun({ text: "Key Activities:", bold: true })] }),
    ...model.keyActivities.map(a => new Paragraph({ children: [new TextRun(`  • ${a}`)] })),
    new Paragraph({ spacing: { before: 150 }, children: [new TextRun({ text: "Key Partnerships:", bold: true })] }),
    ...model.keyPartnerships.map(p => new Paragraph({ children: [new TextRun(`  • ${p}`)] })),
    new Paragraph({ spacing: { before: 150 }, children: [new TextRun({ text: "Cost Structure:", bold: true })] }),
    ...model.costStructure.map(c => new Paragraph({ children: [new TextRun(`  • ${c}`)] }))
  ];
  
  return sections;
}

// Generate document
Packer.toBuffer(doc).then(buffer => {
  fs.writeFileSync('/home/claude/vendorplatform/docs/Vendor_Platform_Strategy_Document.docx', buffer);
  console.log('Document created successfully!');
});
