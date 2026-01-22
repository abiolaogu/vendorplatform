-- ============================================================================
-- VENDOR & ARTISANS PLATFORM - SEED DATA
-- Comprehensive adjacency mappings for all service clusters
-- ============================================================================

-- ============================================================================
-- SECTION 1: LIFE EVENT TRIGGERS
-- ============================================================================

INSERT INTO life_event_triggers (id, name, slug, code, event_type, cluster_type, description, typical_timeline_days, peak_months, avg_services_booked, avg_spend, avg_lead_time_days) VALUES

-- CELEBRATIONS CLUSTER
('11111111-0001-0001-0001-000000000001', 'Wedding', 'wedding', 'WEDDING', 'celebration', 'celebrations', 'Marriage ceremony and reception planning', 365, ARRAY[11,12,1,2,3,4], 15.0, 5000000, 180),
('11111111-0001-0001-0001-000000000002', 'Engagement', 'engagement', 'ENGAGEMENT', 'milestone', 'celebrations', 'Engagement announcement and party', 30, ARRAY[2,12], 5.0, 500000, 14),
('11111111-0001-0001-0001-000000000003', 'Birthday Party', 'birthday-party', 'BIRTHDAY', 'celebration', 'celebrations', 'Birthday celebration planning', 14, NULL, 6.0, 200000, 7),
('11111111-0001-0001-0001-000000000004', 'Anniversary', 'anniversary', 'ANNIVERSARY', 'celebration', 'celebrations', 'Wedding or relationship anniversary', 30, NULL, 4.0, 300000, 14),
('11111111-0001-0001-0001-000000000005', 'Funeral/Memorial', 'funeral-memorial', 'FUNERAL', 'transition', 'celebrations', 'End of life services coordination', 7, NULL, 10.0, 1500000, 3),
('11111111-0001-0001-0001-000000000006', 'Baby Shower', 'baby-shower', 'BABY_SHOWER', 'celebration', 'celebrations', 'Pre-birth celebration', 30, NULL, 5.0, 150000, 14),
('11111111-0001-0001-0001-000000000007', 'Naming Ceremony', 'naming-ceremony', 'NAMING', 'celebration', 'celebrations', 'Baby naming and dedication', 14, NULL, 8.0, 400000, 7),
('11111111-0001-0001-0001-000000000008', 'Graduation', 'graduation', 'GRADUATION', 'milestone', 'celebrations', 'Academic graduation celebration', 30, ARRAY[6,7,11,12], 5.0, 250000, 14),
('11111111-0001-0001-0001-000000000009', 'Housewarming', 'housewarming', 'HOUSEWARMING', 'celebration', 'celebrations', 'New home celebration', 14, NULL, 4.0, 200000, 7),

-- HOME CLUSTER
('11111111-0001-0002-0001-000000000001', 'Home Purchase', 'home-purchase', 'HOME_PURCHASE', 'transition', 'home', 'Buying a new home', 90, NULL, 12.0, 2000000, 30),
('11111111-0001-0002-0001-000000000002', 'Relocation/Moving', 'relocation-moving', 'RELOCATION', 'transition', 'home', 'Moving to a new residence', 30, ARRAY[12,1,6,7], 8.0, 500000, 14),
('11111111-0001-0002-0001-000000000003', 'Home Renovation', 'home-renovation', 'RENOVATION', 'transition', 'home', 'Major home improvement project', 90, NULL, 10.0, 3000000, 30),
('11111111-0001-0002-0001-000000000004', 'Home Emergency', 'home-emergency', 'HOME_EMERGENCY', 'emergency', 'home', 'Urgent home repair needs', 1, NULL, 3.0, 100000, 0),
('11111111-0001-0002-0001-000000000005', 'Seasonal Maintenance', 'seasonal-maintenance', 'SEASONAL_MAINT', 'routine', 'home', 'Regular home maintenance', 7, ARRAY[3,4,9,10], 4.0, 150000, 7),

-- TRAVEL CLUSTER
('11111111-0001-0003-0001-000000000001', 'Domestic Flight', 'domestic-flight', 'DOMESTIC_FLIGHT', 'transition', 'travel', 'Air travel within country', 7, ARRAY[12,1,4,8], 4.0, 150000, 7),
('11111111-0001-0003-0001-000000000002', 'International Travel', 'international-travel', 'INTL_TRAVEL', 'transition', 'travel', 'International travel planning', 60, ARRAY[6,7,8,12], 8.0, 1500000, 30),
('11111111-0001-0003-0001-000000000003', 'Business Trip', 'business-trip', 'BUSINESS_TRIP', 'routine', 'travel', 'Work-related travel', 7, NULL, 4.0, 300000, 3),
('11111111-0001-0003-0001-000000000004', 'Relocation Abroad', 'relocation-abroad', 'RELOCATION_ABROAD', 'transition', 'travel', 'Moving to another country', 180, NULL, 15.0, 5000000, 90),
('11111111-0001-0003-0001-000000000005', 'Vacation/Holiday', 'vacation-holiday', 'VACATION', 'celebration', 'travel', 'Leisure travel planning', 30, ARRAY[4,6,7,8,12], 6.0, 500000, 14),

-- HORECA CLUSTER
('11111111-0001-0004-0001-000000000001', 'Restaurant Launch', 'restaurant-launch', 'REST_LAUNCH', 'transition', 'horeca', 'Opening a new restaurant', 180, NULL, 20.0, 15000000, 90),
('11111111-0001-0004-0001-000000000002', 'Corporate Event', 'corporate-event', 'CORP_EVENT', 'celebration', 'horeca', 'Business event planning', 60, NULL, 12.0, 2000000, 30),
('11111111-0001-0004-0001-000000000003', 'Private Chef Booking', 'private-chef', 'PRIVATE_CHEF', 'routine', 'horeca', 'Home dining experience', 7, NULL, 3.0, 100000, 3),
('11111111-0001-0004-0001-000000000004', 'Food Business Launch', 'food-business-launch', 'FOOD_BIZ_LAUNCH', 'transition', 'horeca', 'Starting a food business', 90, NULL, 15.0, 2000000, 60),

-- FASHION & PERSONAL CARE CLUSTER
('11111111-0001-0005-0001-000000000001', 'Personal Makeover', 'personal-makeover', 'MAKEOVER', 'milestone', 'fashion', 'Complete personal transformation', 30, ARRAY[1,12], 8.0, 500000, 14),
('11111111-0001-0005-0001-000000000002', 'Fashion Brand Launch', 'fashion-brand-launch', 'FASHION_LAUNCH', 'transition', 'fashion', 'Starting a fashion business', 180, NULL, 12.0, 3000000, 90),
('11111111-0001-0005-0001-000000000003', 'Bridal Styling', 'bridal-styling', 'BRIDAL_STYLE', 'celebration', 'fashion', 'Complete bridal look preparation', 60, ARRAY[11,12,1,2,3,4], 6.0, 800000, 30),

-- BUSINESS CLUSTER
('11111111-0001-0006-0001-000000000001', 'Business Launch', 'business-launch', 'BIZ_LAUNCH', 'transition', 'business', 'Starting a new company', 90, ARRAY[1], 15.0, 2000000, 60),
('11111111-0001-0006-0001-000000000002', 'Office Setup', 'office-setup', 'OFFICE_SETUP', 'transition', 'business', 'Setting up office space', 60, NULL, 10.0, 3000000, 30),
('11111111-0001-0006-0001-000000000003', 'Product Launch Event', 'product-launch-event', 'PRODUCT_LAUNCH', 'celebration', 'business', 'New product introduction', 60, NULL, 12.0, 2500000, 30),

-- EDUCATION CLUSTER
('11111111-0001-0007-0001-000000000001', 'School Enrollment', 'school-enrollment', 'SCHOOL_ENROLL', 'transition', 'education', 'New school year preparation', 60, ARRAY[8,9], 5.0, 500000, 30),
('11111111-0001-0007-0001-000000000002', 'Study Abroad', 'study-abroad', 'STUDY_ABROAD', 'transition', 'education', 'International education', 180, ARRAY[1,8,9], 12.0, 3000000, 90),
('11111111-0001-0007-0001-000000000003', 'School Setup', 'school-setup', 'SCHOOL_SETUP', 'transition', 'education', 'Establishing a new school', 365, NULL, 25.0, 50000000, 180),

-- HEALTH CLUSTER
('11111111-0001-0008-0001-000000000001', 'Medical Procedure', 'medical-procedure', 'MEDICAL_PROC', 'transition', 'health', 'Scheduled medical treatment', 30, NULL, 6.0, 1000000, 14),
('11111111-0001-0008-0001-000000000002', 'Elderly Care Setup', 'elderly-care-setup', 'ELDERLY_CARE', 'transition', 'health', 'Arranging care for aging parents', 30, NULL, 8.0, 500000, 14),
('11111111-0001-0008-0001-000000000003', 'Fitness Journey', 'fitness-journey', 'FITNESS', 'milestone', 'health', 'Personal health transformation', 90, ARRAY[1], 5.0, 300000, 7),
('11111111-0001-0008-0001-000000000004', 'Childbirth', 'childbirth', 'CHILDBIRTH', 'transition', 'health', 'Pregnancy and delivery', 270, NULL, 10.0, 1500000, 180),

-- AUTOMOTIVE CLUSTER
('11111111-0001-0009-0001-000000000001', 'Vehicle Purchase', 'vehicle-purchase', 'VEH_PURCHASE', 'transition', 'automotive', 'Buying a new vehicle', 30, NULL, 6.0, 500000, 14),
('11111111-0001-0009-0001-000000000002', 'Vehicle Accident', 'vehicle-accident', 'VEH_ACCIDENT', 'emergency', 'automotive', 'Post-accident services', 7, NULL, 5.0, 300000, 0),
('11111111-0001-0009-0001-000000000003', 'Fleet Setup', 'fleet-setup', 'FLEET_SETUP', 'transition', 'automotive', 'Business vehicle fleet', 60, NULL, 10.0, 5000000, 30),

-- CREATIVE CLUSTER
('11111111-0001-0010-0001-000000000001', 'Content Production', 'content-production', 'CONTENT_PROD', 'routine', 'creative', 'Marketing content creation', 14, NULL, 6.0, 500000, 7),
('11111111-0001-0010-0001-000000000002', 'Art Exhibition', 'art-exhibition', 'ART_EXHIBIT', 'celebration', 'creative', 'Art show planning', 90, NULL, 10.0, 1000000, 60),
('11111111-0001-0010-0001-000000000003', 'Music Production', 'music-production', 'MUSIC_PROD', 'milestone', 'creative', 'Album or single production', 90, NULL, 8.0, 1500000, 60),

-- PROPERTY CLUSTER
('11111111-0001-0011-0001-000000000001', 'Property Development', 'property-development', 'PROP_DEV', 'transition', 'property', 'Building construction project', 730, NULL, 20.0, 100000000, 365),
('11111111-0001-0011-0001-000000000002', 'Property Sale', 'property-sale', 'PROP_SALE', 'transition', 'property', 'Selling real estate', 90, NULL, 8.0, 1000000, 30),
('11111111-0001-0011-0001-000000000003', 'Rental Property Setup', 'rental-property-setup', 'RENTAL_SETUP', 'transition', 'property', 'Preparing property for rent', 30, NULL, 6.0, 500000, 14),

-- ENERGY CLUSTER
('11111111-0001-0012-0001-000000000001', 'Solar Installation', 'solar-installation', 'SOLAR_INSTALL', 'transition', 'energy', 'Home solar power setup', 30, NULL, 5.0, 2000000, 14),
('11111111-0001-0012-0001-000000000002', 'Generator Setup', 'generator-setup', 'GEN_SETUP', 'transition', 'energy', 'Backup power installation', 7, NULL, 4.0, 500000, 3),

-- SECURITY CLUSTER
('11111111-0001-0013-0001-000000000001', 'Home Security Setup', 'home-security-setup', 'HOME_SECURITY', 'transition', 'security', 'Residential security installation', 14, NULL, 5.0, 500000, 7),
('11111111-0001-0013-0001-000000000002', 'Event Security', 'event-security', 'EVENT_SECURITY', 'routine', 'security', 'Security for gatherings', 7, NULL, 4.0, 200000, 3),

-- PET CLUSTER
('11111111-0001-0014-0001-000000000001', 'Pet Adoption', 'pet-adoption', 'PET_ADOPT', 'transition', 'pet', 'Getting a new pet', 14, NULL, 6.0, 100000, 7),
('11111111-0001-0014-0001-000000000002', 'Pet Travel', 'pet-travel', 'PET_TRAVEL', 'routine', 'pet', 'Traveling with pets', 14, NULL, 4.0, 150000, 7);

-- ============================================================================
-- SECTION 2: SERVICE CATEGORIES (Hierarchical)
-- ============================================================================

-- Level 0: Clusters
INSERT INTO service_categories (id, parent_id, level, path, name, slug, code, cluster_type, short_description) VALUES
('22222222-0000-0001-0000-000000000001', NULL, 0, 'celebrations', 'Celebrations & Events', 'celebrations-events', 'CELEBRATIONS', 'celebrations', 'Wedding, parties, and milestone celebrations'),
('22222222-0000-0002-0000-000000000001', NULL, 0, 'home', 'Home & Property Services', 'home-property-services', 'HOME', 'home', 'Home improvement, maintenance, and relocation'),
('22222222-0000-0003-0000-000000000001', NULL, 0, 'travel', 'Travel & Mobility', 'travel-mobility', 'TRAVEL', 'travel', 'Transportation, accommodation, and travel services'),
('22222222-0000-0004-0000-000000000001', NULL, 0, 'horeca', 'Food & Hospitality', 'food-hospitality', 'HORECA', 'horeca', 'Restaurants, catering, and food services'),
('22222222-0000-0005-0000-000000000001', NULL, 0, 'fashion', 'Fashion & Personal Care', 'fashion-personal-care', 'FASHION', 'fashion', 'Styling, grooming, and fashion services'),
('22222222-0000-0006-0000-000000000001', NULL, 0, 'business', 'Business Services', 'business-services', 'BUSINESS', 'business', 'Corporate events, office setup, and business support'),
('22222222-0000-0007-0000-000000000001', NULL, 0, 'education', 'Education & Learning', 'education-learning', 'EDUCATION', 'education', 'Schools, tutoring, and educational services'),
('22222222-0000-0008-0000-000000000001', NULL, 0, 'health', 'Health & Wellness', 'health-wellness', 'HEALTH', 'health', 'Medical, fitness, and wellness services'),
('22222222-0000-0009-0000-000000000001', NULL, 0, 'automotive', 'Automotive', 'automotive', 'AUTO', 'automotive', 'Vehicle sales, repair, and maintenance'),
('22222222-0000-0010-0000-000000000001', NULL, 0, 'creative', 'Creative & Content', 'creative-content', 'CREATIVE', 'creative', 'Photography, video, and creative services'),
('22222222-0000-0011-0000-000000000001', NULL, 0, 'property', 'Property & Construction', 'property-construction', 'PROPERTY', 'property', 'Real estate and construction services'),
('22222222-0000-0012-0000-000000000001', NULL, 0, 'energy', 'Energy & Utilities', 'energy-utilities', 'ENERGY', 'energy', 'Power, solar, and utility services'),
('22222222-0000-0013-0000-000000000001', NULL, 0, 'security', 'Security Services', 'security-services', 'SECURITY', 'security', 'Safety, security, and protection'),
('22222222-0000-0014-0000-000000000001', NULL, 0, 'pet', 'Pet & Animal Care', 'pet-animal-care', 'PET', 'pet', 'Pet services and animal care');

-- Level 1: Categories under CELEBRATIONS
INSERT INTO service_categories (id, parent_id, level, path, name, slug, code, cluster_type) VALUES
-- Venue & Space
('22222222-0001-0001-0001-000000000001', '22222222-0000-0001-0000-000000000001', 1, 'celebrations.venue', 'Event Venues', 'event-venues', 'VENUE', 'celebrations'),
('22222222-0001-0001-0002-000000000001', '22222222-0000-0001-0000-000000000001', 1, 'celebrations.catering', 'Catering', 'catering', 'CATERING', 'celebrations'),
('22222222-0001-0001-0003-000000000001', '22222222-0000-0001-0000-000000000001', 1, 'celebrations.decoration', 'Event Decoration', 'event-decoration', 'DECORATION', 'celebrations'),
('22222222-0001-0001-0004-000000000001', '22222222-0000-0001-0000-000000000001', 1, 'celebrations.photography', 'Event Photography', 'event-photography', 'PHOTO', 'celebrations'),
('22222222-0001-0001-0005-000000000001', '22222222-0000-0001-0000-000000000001', 1, 'celebrations.videography', 'Event Videography', 'event-videography', 'VIDEO', 'celebrations'),
('22222222-0001-0001-0006-000000000001', '22222222-0000-0001-0000-000000000001', 1, 'celebrations.entertainment', 'Entertainment & DJ', 'entertainment-dj', 'ENTERTAIN', 'celebrations'),
('22222222-0001-0001-0007-000000000001', '22222222-0000-0001-0000-000000000001', 1, 'celebrations.mc', 'MC & Hosting', 'mc-hosting', 'MC', 'celebrations'),
('22222222-0001-0001-0008-000000000001', '22222222-0000-0001-0000-000000000001', 1, 'celebrations.cake', 'Cakes & Confectionery', 'cakes-confectionery', 'CAKE', 'celebrations'),
('22222222-0001-0001-0009-000000000001', '22222222-0000-0001-0000-000000000001', 1, 'celebrations.florist', 'Florists', 'florists', 'FLORIST', 'celebrations'),
('22222222-0001-0001-0010-000000000001', '22222222-0000-0001-0000-000000000001', 1, 'celebrations.makeup', 'Makeup & Styling', 'makeup-styling', 'MAKEUP', 'celebrations'),
('22222222-0001-0001-0011-000000000001', '22222222-0000-0001-0000-000000000001', 1, 'celebrations.fashion', 'Event Fashion', 'event-fashion', 'EVENT_FASHION', 'celebrations'),
('22222222-0001-0001-0012-000000000001', '22222222-0000-0001-0000-000000000001', 1, 'celebrations.transport', 'Event Transport', 'event-transport', 'EVENT_TRANSPORT', 'celebrations'),
('22222222-0001-0001-0013-000000000001', '22222222-0000-0001-0000-000000000001', 1, 'celebrations.lighting', 'Event Lighting', 'event-lighting', 'LIGHTING', 'celebrations'),
('22222222-0001-0001-0014-000000000001', '22222222-0000-0001-0000-000000000001', 1, 'celebrations.sound', 'Sound & PA Systems', 'sound-pa-systems', 'SOUND', 'celebrations'),
('22222222-0001-0001-0015-000000000001', '22222222-0000-0001-0000-000000000001', 1, 'celebrations.stationery', 'Invitations & Stationery', 'invitations-stationery', 'STATIONERY', 'celebrations'),
('22222222-0001-0001-0016-000000000001', '22222222-0000-0001-0000-000000000001', 1, 'celebrations.planner', 'Event Planners', 'event-planners', 'PLANNER', 'celebrations'),
('22222222-0001-0001-0017-000000000001', '22222222-0000-0001-0000-000000000001', 1, 'celebrations.security', 'Event Security', 'event-security', 'EVENT_SECURITY', 'celebrations'),
('22222222-0001-0001-0018-000000000001', '22222222-0000-0001-0000-000000000001', 1, 'celebrations.equipment', 'Equipment Rental', 'equipment-rental', 'EQUIPMENT', 'celebrations'),
('22222222-0001-0001-0019-000000000001', '22222222-0000-0001-0000-000000000001', 1, 'celebrations.ushers', 'Ushers & Waitstaff', 'ushers-waitstaff', 'USHERS', 'celebrations'),
('22222222-0001-0001-0020-000000000001', '22222222-0000-0001-0000-000000000001', 1, 'celebrations.traditional', 'Traditional Performers', 'traditional-performers', 'TRADITIONAL', 'celebrations'),
('22222222-0001-0001-0021-000000000001', '22222222-0000-0001-0000-000000000001', 1, 'celebrations.kids', 'Kids Entertainment', 'kids-entertainment', 'KIDS_ENTERTAIN', 'celebrations'),
('22222222-0001-0001-0022-000000000001', '22222222-0000-0001-0000-000000000001', 1, 'celebrations.drinks', 'Drinks & Bartending', 'drinks-bartending', 'DRINKS', 'celebrations'),
('22222222-0001-0001-0023-000000000001', '22222222-0000-0001-0000-000000000001', 1, 'celebrations.gifts', 'Gift Services', 'gift-services', 'GIFTS', 'celebrations'),
('22222222-0001-0001-0024-000000000001', '22222222-0000-0001-0000-000000000001', 1, 'celebrations.officiant', 'Religious Officiants', 'religious-officiants', 'OFFICIANT', 'celebrations'),
('22222222-0001-0001-0025-000000000001', '22222222-0000-0001-0000-000000000001', 1, 'celebrations.cleanup', 'Event Cleanup', 'event-cleanup', 'CLEANUP', 'celebrations'),

-- Funeral specific
('22222222-0001-0001-0026-000000000001', '22222222-0000-0001-0000-000000000001', 1, 'celebrations.mortuary', 'Mortuary Services', 'mortuary-services', 'MORTUARY', 'celebrations'),
('22222222-0001-0001-0027-000000000001', '22222222-0000-0001-0000-000000000001', 1, 'celebrations.casket', 'Caskets & Urns', 'caskets-urns', 'CASKET', 'celebrations'),
('22222222-0001-0001-0028-000000000001', '22222222-0000-0001-0000-000000000001', 1, 'celebrations.memorial', 'Memorial Services', 'memorial-services', 'MEMORIAL', 'celebrations');

-- Level 1: Categories under HOME
INSERT INTO service_categories (id, parent_id, level, path, name, slug, code, cluster_type) VALUES
('22222222-0001-0002-0001-000000000001', '22222222-0000-0002-0000-000000000001', 1, 'home.moving', 'Moving & Relocation', 'moving-relocation', 'MOVING', 'home'),
('22222222-0001-0002-0002-000000000001', '22222222-0000-0002-0000-000000000001', 1, 'home.cleaning', 'Cleaning Services', 'cleaning-services', 'CLEANING', 'home'),
('22222222-0001-0002-0003-000000000001', '22222222-0000-0002-0000-000000000001', 1, 'home.plumbing', 'Plumbing', 'plumbing', 'PLUMBING', 'home'),
('22222222-0001-0002-0004-000000000001', '22222222-0000-0002-0000-000000000001', 1, 'home.electrical', 'Electrical', 'electrical', 'ELECTRICAL', 'home'),
('22222222-0001-0002-0005-000000000001', '22222222-0000-0002-0000-000000000001', 1, 'home.painting', 'Painting', 'painting', 'PAINTING', 'home'),
('22222222-0001-0002-0006-000000000001', '22222222-0000-0002-0000-000000000001', 1, 'home.carpentry', 'Carpentry', 'carpentry', 'CARPENTRY', 'home'),
('22222222-0001-0002-0007-000000000001', '22222222-0000-0002-0000-000000000001', 1, 'home.tiling', 'Tiling & Flooring', 'tiling-flooring', 'TILING', 'home'),
('22222222-0001-0002-0008-000000000001', '22222222-0000-0002-0000-000000000001', 1, 'home.ac', 'AC & HVAC', 'ac-hvac', 'AC_HVAC', 'home'),
('22222222-0001-0002-0009-000000000001', '22222222-0000-0002-0000-000000000001', 1, 'home.interior', 'Interior Design', 'interior-design', 'INTERIOR', 'home'),
('22222222-0001-0002-0010-000000000001', '22222222-0000-0002-0000-000000000001', 1, 'home.landscaping', 'Landscaping', 'landscaping', 'LANDSCAPING', 'home'),
('22222222-0001-0002-0011-000000000001', '22222222-0000-0002-0000-000000000001', 1, 'home.pest', 'Pest Control', 'pest-control', 'PEST', 'home'),
('22222222-0001-0002-0012-000000000001', '22222222-0000-0002-0000-000000000001', 1, 'home.roofing', 'Roofing', 'roofing', 'ROOFING', 'home'),
('22222222-0001-0002-0013-000000000001', '22222222-0000-0002-0000-000000000001', 1, 'home.appliance', 'Appliance Repair', 'appliance-repair', 'APPLIANCE', 'home'),
('22222222-0001-0002-0014-000000000001', '22222222-0000-0002-0000-000000000001', 1, 'home.furniture', 'Furniture Assembly', 'furniture-assembly', 'FURNITURE', 'home'),
('22222222-0001-0002-0015-000000000001', '22222222-0000-0002-0000-000000000001', 1, 'home.curtains', 'Curtains & Blinds', 'curtains-blinds', 'CURTAINS', 'home'),
('22222222-0001-0002-0016-000000000001', '22222222-0000-0002-0000-000000000001', 1, 'home.security', 'Home Security', 'home-security', 'HOME_SECURITY', 'home'),
('22222222-0001-0002-0017-000000000001', '22222222-0000-0002-0000-000000000001', 1, 'home.pool', 'Pool Services', 'pool-services', 'POOL', 'home'),
('22222222-0001-0002-0018-000000000001', '22222222-0000-0002-0000-000000000001', 1, 'home.waterproof', 'Waterproofing', 'waterproofing', 'WATERPROOF', 'home'),
('22222222-0001-0002-0019-000000000001', '22222222-0000-0002-0000-000000000001', 1, 'home.glass', 'Glass & Glazing', 'glass-glazing', 'GLASS', 'home'),
('22222222-0001-0002-0020-000000000001', '22222222-0000-0002-0000-000000000001', 1, 'home.welding', 'Welding & Metalwork', 'welding-metalwork', 'WELDING', 'home'),
('22222222-0001-0002-0021-000000000001', '22222222-0000-0002-0000-000000000001', 1, 'home.smarthome', 'Smart Home Installation', 'smart-home-installation', 'SMARTHOME', 'home');

-- Level 1: Categories under TRAVEL
INSERT INTO service_categories (id, parent_id, level, path, name, slug, code, cluster_type) VALUES
('22222222-0001-0003-0001-000000000001', '22222222-0000-0003-0000-000000000001', 1, 'travel.taxi', 'Taxi & Ride Services', 'taxi-ride-services', 'TAXI', 'travel'),
('22222222-0001-0003-0002-000000000001', '22222222-0000-0003-0000-000000000001', 1, 'travel.carrental', 'Car Rental', 'car-rental', 'CAR_RENTAL', 'travel'),
('22222222-0001-0003-0003-000000000001', '22222222-0000-0003-0000-000000000001', 1, 'travel.hotel', 'Hotels & Accommodation', 'hotels-accommodation', 'HOTEL', 'travel'),
('22222222-0001-0003-0004-000000000001', '22222222-0000-0003-0000-000000000001', 1, 'travel.airport', 'Airport Services', 'airport-services', 'AIRPORT', 'travel'),
('22222222-0001-0003-0005-000000000001', '22222222-0000-0003-0000-000000000001', 1, 'travel.visa', 'Visa & Immigration', 'visa-immigration', 'VISA', 'travel'),
('22222222-0001-0003-0006-000000000001', '22222222-0000-0003-0000-000000000001', 1, 'travel.tours', 'Tours & Guides', 'tours-guides', 'TOURS', 'travel'),
('22222222-0001-0003-0007-000000000001', '22222222-0000-0003-0000-000000000001', 1, 'travel.forex', 'Currency Exchange', 'currency-exchange', 'FOREX', 'travel'),
('22222222-0001-0003-0008-000000000001', '22222222-0000-0003-0000-000000000001', 1, 'travel.insurance', 'Travel Insurance', 'travel-insurance', 'TRAVEL_INSURANCE', 'travel'),
('22222222-0001-0003-0009-000000000001', '22222222-0000-0003-0000-000000000001', 1, 'travel.luggage', 'Luggage & Travel Gear', 'luggage-travel-gear', 'LUGGAGE', 'travel'),
('22222222-0001-0003-0010-000000000001', '22222222-0000-0003-0000-000000000001', 1, 'travel.lounge', 'Airport Lounges', 'airport-lounges', 'LOUNGE', 'travel'),
('22222222-0001-0003-0011-000000000001', '22222222-0000-0003-0000-000000000001', 1, 'travel.driver', 'Chauffeur Services', 'chauffeur-services', 'CHAUFFEUR', 'travel'),
('22222222-0001-0003-0012-000000000001', '22222222-0000-0003-0000-000000000001', 1, 'travel.concierge', 'Travel Concierge', 'travel-concierge', 'CONCIERGE', 'travel');

-- Level 1: Categories under HORECA
INSERT INTO service_categories (id, parent_id, level, path, name, slug, code, cluster_type) VALUES
('22222222-0001-0004-0001-000000000001', '22222222-0000-0004-0000-000000000001', 1, 'horeca.privatechef', 'Private Chefs', 'private-chefs', 'PRIVATE_CHEF', 'horeca'),
('22222222-0001-0004-0002-000000000001', '22222222-0000-0004-0000-000000000001', 1, 'horeca.mealprep', 'Meal Prep Services', 'meal-prep-services', 'MEAL_PREP', 'horeca'),
('22222222-0001-0004-0003-000000000001', '22222222-0000-0004-0000-000000000001', 1, 'horeca.restaurant_consult', 'Restaurant Consulting', 'restaurant-consulting', 'REST_CONSULT', 'horeca'),
('22222222-0001-0004-0004-000000000001', '22222222-0000-0004-0000-000000000001', 1, 'horeca.kitchen_equip', 'Kitchen Equipment', 'kitchen-equipment', 'KITCHEN_EQUIP', 'horeca'),
('22222222-0001-0004-0005-000000000001', '22222222-0000-0004-0000-000000000001', 1, 'horeca.food_supply', 'Food Suppliers', 'food-suppliers', 'FOOD_SUPPLY', 'horeca'),
('22222222-0001-0004-0006-000000000001', '22222222-0000-0004-0000-000000000001', 1, 'horeca.nutritionist', 'Nutritionists', 'nutritionists', 'NUTRITIONIST', 'horeca'),
('22222222-0001-0004-0007-000000000001', '22222222-0000-0004-0000-000000000001', 1, 'horeca.food_photo', 'Food Photography', 'food-photography', 'FOOD_PHOTO', 'horeca'),
('22222222-0001-0004-0008-000000000001', '22222222-0000-0004-0000-000000000001', 1, 'horeca.menu_design', 'Menu Design', 'menu-design', 'MENU_DESIGN', 'horeca'),
('22222222-0001-0004-0009-000000000001', '22222222-0000-0004-0000-000000000001', 1, 'horeca.food_license', 'Food Licensing & NAFDAC', 'food-licensing-nafdac', 'FOOD_LICENSE', 'horeca'),
('22222222-0001-0004-0010-000000000001', '22222222-0000-0004-0000-000000000001', 1, 'horeca.cooking_class', 'Cooking Classes', 'cooking-classes', 'COOKING_CLASS', 'horeca');

-- Level 1: Categories under FASHION
INSERT INTO service_categories (id, parent_id, level, path, name, slug, code, cluster_type) VALUES
('22222222-0001-0005-0001-000000000001', '22222222-0000-0005-0000-000000000001', 1, 'fashion.stylist', 'Personal Stylists', 'personal-stylists', 'STYLIST', 'fashion'),
('22222222-0001-0005-0002-000000000001', '22222222-0000-0005-0000-000000000001', 1, 'fashion.tailor', 'Tailors & Seamstresses', 'tailors-seamstresses', 'TAILOR', 'fashion'),
('22222222-0001-0005-0003-000000000001', '22222222-0000-0005-0000-000000000001', 1, 'fashion.hair', 'Hair Stylists', 'hair-stylists', 'HAIR', 'fashion'),
('22222222-0001-0005-0004-000000000001', '22222222-0000-0005-0000-000000000001', 1, 'fashion.barber', 'Barbers', 'barbers', 'BARBER', 'fashion'),
('22222222-0001-0005-0005-000000000001', '22222222-0000-0005-0000-000000000001', 1, 'fashion.nails', 'Nail Technicians', 'nail-technicians', 'NAILS', 'fashion'),
('22222222-0001-0005-0006-000000000001', '22222222-0000-0005-0000-000000000001', 1, 'fashion.skincare', 'Skincare Specialists', 'skincare-specialists', 'SKINCARE', 'fashion'),
('22222222-0001-0005-0007-000000000001', '22222222-0000-0005-0000-000000000001', 1, 'fashion.spa', 'Spa Services', 'spa-services', 'SPA', 'fashion'),
('22222222-0001-0005-0008-000000000001', '22222222-0000-0005-0000-000000000001', 1, 'fashion.jewelry', 'Jewelry & Accessories', 'jewelry-accessories', 'JEWELRY', 'fashion'),
('22222222-0001-0005-0009-000000000001', '22222222-0000-0005-0000-000000000001', 1, 'fashion.henna', 'Henna Artists', 'henna-artists', 'HENNA', 'fashion'),
('22222222-0001-0005-0010-000000000001', '22222222-0000-0005-0000-000000000001', 1, 'fashion.fabric', 'Fabric Vendors', 'fabric-vendors', 'FABRIC', 'fashion'),
('22222222-0001-0005-0011-000000000001', '22222222-0000-0005-0000-000000000001', 1, 'fashion.shoes', 'Shoe Makers & Vendors', 'shoe-makers-vendors', 'SHOES', 'fashion'),
('22222222-0001-0005-0012-000000000001', '22222222-0000-0005-0000-000000000001', 1, 'fashion.fitness_trainer', 'Personal Trainers', 'personal-trainers', 'FITNESS_TRAINER', 'fashion');

-- Level 1: Categories under BUSINESS
INSERT INTO service_categories (id, parent_id, level, path, name, slug, code, cluster_type) VALUES
('22222222-0001-0006-0001-000000000001', '22222222-0000-0006-0000-000000000001', 1, 'business.registration', 'Business Registration', 'business-registration', 'BIZ_REG', 'business'),
('22222222-0001-0006-0002-000000000001', '22222222-0000-0006-0000-000000000001', 1, 'business.legal', 'Legal Services', 'legal-services', 'LEGAL', 'business'),
('22222222-0001-0006-0003-000000000001', '22222222-0000-0006-0000-000000000001', 1, 'business.accounting', 'Accounting & Tax', 'accounting-tax', 'ACCOUNTING', 'business'),
('22222222-0001-0006-0004-000000000001', '22222222-0000-0006-0000-000000000001', 1, 'business.hr', 'HR & Recruitment', 'hr-recruitment', 'HR', 'business'),
('22222222-0001-0006-0005-000000000001', '22222222-0000-0006-0000-000000000001', 1, 'business.branding', 'Branding & Design', 'branding-design', 'BRANDING', 'business'),
('22222222-0001-0006-0006-000000000001', '22222222-0000-0006-0000-000000000001', 1, 'business.webdev', 'Web Development', 'web-development', 'WEBDEV', 'business'),
('22222222-0001-0006-0007-000000000001', '22222222-0000-0006-0000-000000000001', 1, 'business.marketing', 'Marketing & PR', 'marketing-pr', 'MARKETING', 'business'),
('22222222-0001-0006-0008-000000000001', '22222222-0000-0006-0000-000000000001', 1, 'business.it', 'IT Services', 'it-services', 'IT', 'business'),
('22222222-0001-0006-0009-000000000001', '22222222-0000-0006-0000-000000000001', 1, 'business.office_furniture', 'Office Furniture', 'office-furniture', 'OFFICE_FURN', 'business'),
('22222222-0001-0006-0010-000000000001', '22222222-0000-0006-0000-000000000001', 1, 'business.printing', 'Printing & Stationery', 'printing-stationery', 'PRINTING', 'business'),
('22222222-0001-0006-0011-000000000001', '22222222-0000-0006-0000-000000000001', 1, 'business.insurance', 'Business Insurance', 'business-insurance', 'BIZ_INSURANCE', 'business'),
('22222222-0001-0006-0012-000000000001', '22222222-0000-0006-0000-000000000001', 1, 'business.consulting', 'Business Consulting', 'business-consulting', 'CONSULTING', 'business');

-- Level 1: Categories under HEALTH
INSERT INTO service_categories (id, parent_id, level, path, name, slug, code, cluster_type) VALUES
('22222222-0001-0008-0001-000000000001', '22222222-0000-0008-0000-000000000001', 1, 'health.home_nursing', 'Home Nursing', 'home-nursing', 'HOME_NURSING', 'health'),
('22222222-0001-0008-0002-000000000001', '22222222-0000-0008-0000-000000000001', 1, 'health.physio', 'Physiotherapy', 'physiotherapy', 'PHYSIO', 'health'),
('22222222-0001-0008-0003-000000000001', '22222222-0000-0008-0000-000000000001', 1, 'health.caregivers', 'Caregivers', 'caregivers', 'CAREGIVERS', 'health'),
('22222222-0001-0008-0004-000000000001', '22222222-0000-0008-0000-000000000001', 1, 'health.pharmacy', 'Pharmacy Delivery', 'pharmacy-delivery', 'PHARMACY', 'health'),
('22222222-0001-0008-0005-000000000001', '22222222-0000-0008-0000-000000000001', 1, 'health.lab', 'Lab Tests at Home', 'lab-tests-at-home', 'LAB', 'health'),
('22222222-0001-0008-0006-000000000001', '22222222-0000-0008-0000-000000000001', 1, 'health.telemedicine', 'Telemedicine', 'telemedicine', 'TELEMEDICINE', 'health'),
('22222222-0001-0008-0007-000000000001', '22222222-0000-0008-0000-000000000001', 1, 'health.medical_equip', 'Medical Equipment', 'medical-equipment', 'MEDICAL_EQUIP', 'health'),
('22222222-0001-0008-0008-000000000001', '22222222-0000-0008-0000-000000000001', 1, 'health.mental_health', 'Mental Health', 'mental-health', 'MENTAL_HEALTH', 'health'),
('22222222-0001-0008-0009-000000000001', '22222222-0000-0008-0000-000000000001', 1, 'health.doula', 'Doulas & Midwives', 'doulas-midwives', 'DOULA', 'health'),
('22222222-0001-0008-0010-000000000001', '22222222-0000-0008-0000-000000000001', 1, 'health.yoga', 'Yoga & Meditation', 'yoga-meditation', 'YOGA', 'health');

-- Level 1: Categories under AUTOMOTIVE
INSERT INTO service_categories (id, parent_id, level, path, name, slug, code, cluster_type) VALUES
('22222222-0001-0009-0001-000000000001', '22222222-0000-0009-0000-000000000001', 1, 'automotive.dealer', 'Car Dealers', 'car-dealers', 'CAR_DEALER', 'automotive'),
('22222222-0001-0009-0002-000000000001', '22222222-0000-0009-0000-000000000001', 1, 'automotive.mechanic', 'Mechanics', 'mechanics', 'MECHANIC', 'automotive'),
('22222222-0001-0009-0003-000000000001', '22222222-0000-0009-0000-000000000001', 1, 'automotive.panel_beater', 'Panel Beaters', 'panel-beaters', 'PANEL_BEATER', 'automotive'),
('22222222-0001-0009-0004-000000000001', '22222222-0000-0009-0000-000000000001', 1, 'automotive.towing', 'Towing Services', 'towing-services', 'TOWING', 'automotive'),
('22222222-0001-0009-0005-000000000001', '22222222-0000-0009-0000-000000000001', 1, 'automotive.car_wash', 'Car Wash & Detailing', 'car-wash-detailing', 'CAR_WASH', 'automotive'),
('22222222-0001-0009-0006-000000000001', '22222222-0000-0009-0000-000000000001', 1, 'automotive.auto_insurance', 'Auto Insurance', 'auto-insurance', 'AUTO_INSURANCE', 'automotive'),
('22222222-0001-0009-0007-000000000001', '22222222-0000-0009-0000-000000000001', 1, 'automotive.spare_parts', 'Spare Parts', 'spare-parts', 'SPARE_PARTS', 'automotive'),
('22222222-0001-0009-0008-000000000001', '22222222-0000-0009-0000-000000000001', 1, 'automotive.tracker', 'Vehicle Tracking', 'vehicle-tracking', 'TRACKER', 'automotive'),
('22222222-0001-0009-0009-000000000001', '22222222-0000-0009-0000-000000000001', 1, 'automotive.tinting', 'Tinting & Wrapping', 'tinting-wrapping', 'TINTING', 'automotive'),
('22222222-0001-0009-0010-000000000001', '22222222-0000-0009-0000-000000000001', 1, 'automotive.driver_hire', 'Driver Services', 'driver-services', 'DRIVER_HIRE', 'automotive');

-- Level 1: Categories under CREATIVE
INSERT INTO service_categories (id, parent_id, level, path, name, slug, code, cluster_type) VALUES
('22222222-0001-0010-0001-000000000001', '22222222-0000-0010-0000-000000000001', 1, 'creative.photographer', 'Photographers', 'photographers', 'PHOTOGRAPHER', 'creative'),
('22222222-0001-0010-0002-000000000001', '22222222-0000-0010-0000-000000000001', 1, 'creative.videographer', 'Videographers', 'videographers', 'VIDEOGRAPHER', 'creative'),
('22222222-0001-0010-0003-000000000001', '22222222-0000-0010-0000-000000000001', 1, 'creative.graphic_design', 'Graphic Designers', 'graphic-designers', 'GRAPHIC_DESIGN', 'creative'),
('22222222-0001-0010-0004-000000000001', '22222222-0000-0010-0000-000000000001', 1, 'creative.copywriter', 'Copywriters', 'copywriters', 'COPYWRITER', 'creative'),
('22222222-0001-0010-0005-000000000001', '22222222-0000-0010-0000-000000000001', 1, 'creative.voice_artist', 'Voice Artists', 'voice-artists', 'VOICE_ARTIST', 'creative'),
('22222222-0001-0010-0006-000000000001', '22222222-0000-0010-0000-000000000001', 1, 'creative.music_producer', 'Music Producers', 'music-producers', 'MUSIC_PRODUCER', 'creative'),
('22222222-0001-0010-0007-000000000001', '22222222-0000-0010-0000-000000000001', 1, 'creative.social_media', 'Social Media Managers', 'social-media-managers', 'SOCIAL_MEDIA', 'creative'),
('22222222-0001-0010-0008-000000000001', '22222222-0000-0010-0000-000000000001', 1, 'creative.influencer', 'Influencers', 'influencers', 'INFLUENCER', 'creative'),
('22222222-0001-0010-0009-000000000001', '22222222-0000-0010-0000-000000000001', 1, 'creative.animator', 'Animators', 'animators', 'ANIMATOR', 'creative'),
('22222222-0001-0010-0010-000000000001', '22222222-0000-0010-0000-000000000001', 1, 'creative.editor', 'Video Editors', 'video-editors', 'EDITOR', 'creative');

-- Level 1: Categories under PROPERTY
INSERT INTO service_categories (id, parent_id, level, path, name, slug, code, cluster_type) VALUES
('22222222-0001-0011-0001-000000000001', '22222222-0000-0011-0000-000000000001', 1, 'property.architect', 'Architects', 'architects', 'ARCHITECT', 'property'),
('22222222-0001-0011-0002-000000000001', '22222222-0000-0011-0000-000000000001', 1, 'property.surveyor', 'Surveyors', 'surveyors', 'SURVEYOR', 'property'),
('22222222-0001-0011-0003-000000000001', '22222222-0000-0011-0000-000000000001', 1, 'property.contractor', 'Building Contractors', 'building-contractors', 'CONTRACTOR', 'property'),
('22222222-0001-0011-0004-000000000001', '22222222-0000-0011-0000-000000000001', 1, 'property.realtor', 'Real Estate Agents', 'real-estate-agents', 'REALTOR', 'property'),
('22222222-0001-0011-0005-000000000001', '22222222-0000-0011-0000-000000000001', 1, 'property.quantity_surveyor', 'Quantity Surveyors', 'quantity-surveyors', 'QS', 'property'),
('22222222-0001-0011-0006-000000000001', '22222222-0000-0011-0000-000000000001', 1, 'property.structural_eng', 'Structural Engineers', 'structural-engineers', 'STRUCT_ENG', 'property'),
('22222222-0001-0011-0007-000000000001', '22222222-0000-0011-0000-000000000001', 1, 'property.building_material', 'Building Materials', 'building-materials', 'BUILD_MAT', 'property'),
('22222222-0001-0011-0008-000000000001', '22222222-0000-0011-0000-000000000001', 1, 'property.property_manager', 'Property Managers', 'property-managers', 'PROP_MANAGER', 'property'),
('22222222-0001-0011-0009-000000000001', '22222222-0000-0011-0000-000000000001', 1, 'property.valuer', 'Property Valuers', 'property-valuers', 'VALUER', 'property'),
('22222222-0001-0011-0010-000000000001', '22222222-0000-0011-0000-000000000001', 1, 'property.conveyancer', 'Conveyancing Lawyers', 'conveyancing-lawyers', 'CONVEYANCER', 'property');

-- Level 1: Categories under ENERGY
INSERT INTO service_categories (id, parent_id, level, path, name, slug, code, cluster_type) VALUES
('22222222-0001-0012-0001-000000000001', '22222222-0000-0012-0000-000000000001', 1, 'energy.solar', 'Solar Installation', 'solar-installation', 'SOLAR', 'energy'),
('22222222-0001-0012-0002-000000000001', '22222222-0000-0012-0000-000000000001', 1, 'energy.generator', 'Generator Services', 'generator-services', 'GENERATOR', 'energy'),
('22222222-0001-0012-0003-000000000001', '22222222-0000-0012-0000-000000000001', 1, 'energy.inverter', 'Inverter & Battery', 'inverter-battery', 'INVERTER', 'energy'),
('22222222-0001-0012-0004-000000000001', '22222222-0000-0012-0000-000000000001', 1, 'energy.fuel_delivery', 'Fuel Delivery', 'fuel-delivery', 'FUEL_DELIVERY', 'energy'),
('22222222-0001-0012-0005-000000000001', '22222222-0000-0012-0000-000000000001', 1, 'energy.energy_audit', 'Energy Auditors', 'energy-auditors', 'ENERGY_AUDIT', 'energy'),
('22222222-0001-0012-0006-000000000001', '22222222-0000-0012-0000-000000000001', 1, 'energy.lpg', 'Gas/LPG Services', 'gas-lpg-services', 'LPG', 'energy');

-- ============================================================================
-- SECTION 3: SERVICE ADJACENCIES (Core Recommendation Data)
-- ============================================================================

-- WEDDING ADJACENCIES (Primary trigger: Wedding)
INSERT INTO service_adjacencies (source_category_id, target_category_id, adjacency_type, trigger_context, base_affinity_score, cross_sell_priority, recommendation_copy) VALUES
-- Venue leads to everything
('22222222-0001-0001-0001-000000000001', '22222222-0001-0001-0002-000000000001', 'complementary', 'wedding', 0.95, 95, 'Venues typically require catering services'),
('22222222-0001-0001-0001-000000000001', '22222222-0001-0001-0003-000000000001', 'complementary', 'wedding', 0.92, 92, 'Complete your venue with stunning decoration'),
('22222222-0001-0001-0001-000000000001', '22222222-0001-0001-0004-000000000001', 'complementary', 'wedding', 0.90, 90, 'Capture every moment with professional photography'),
('22222222-0001-0001-0001-000000000001', '22222222-0001-0001-0005-000000000001', 'complementary', 'wedding', 0.88, 88, 'Create lasting memories with videography'),
('22222222-0001-0001-0001-000000000001', '22222222-0001-0001-0006-000000000001', 'complementary', 'wedding', 0.85, 85, 'Keep guests entertained with DJ services'),
('22222222-0001-0001-0001-000000000001', '22222222-0001-0001-0013-000000000001', 'complementary', 'wedding', 0.80, 80, 'Professional lighting transforms any venue'),
('22222222-0001-0001-0001-000000000001', '22222222-0001-0001-0014-000000000001', 'complementary', 'wedding', 0.78, 78, 'Ensure crystal clear sound throughout'),

-- Catering adjacencies
('22222222-0001-0001-0002-000000000001', '22222222-0001-0001-0008-000000000001', 'complementary', 'wedding', 0.90, 90, 'No celebration is complete without a stunning cake'),
('22222222-0001-0001-0002-000000000001', '22222222-0001-0001-0022-000000000001', 'complementary', 'wedding', 0.85, 85, 'Complement your menu with premium drinks'),
('22222222-0001-0001-0002-000000000001', '22222222-0001-0001-0019-000000000001', 'complementary', 'wedding', 0.80, 80, 'Professional waitstaff ensures smooth service'),
('22222222-0001-0001-0002-000000000001', '22222222-0001-0001-0018-000000000001', 'complementary', 'wedding', 0.75, 75, 'Quality tables, chairs, and chinaware'),

-- Makeup/Styling adjacencies
('22222222-0001-0001-0010-000000000001', '22222222-0001-0005-0003-000000000001', 'complementary', 'wedding', 0.92, 92, 'Complete your bridal look with hair styling'),
('22222222-0001-0001-0010-000000000001', '22222222-0001-0005-0005-000000000001', 'complementary', 'wedding', 0.85, 85, 'Beautiful nails complete the perfect look'),
('22222222-0001-0001-0010-000000000001', '22222222-0001-0005-0009-000000000001', 'complementary', 'wedding', 0.80, 80, 'Add traditional henna artistry'),
('22222222-0001-0001-0010-000000000001', '22222222-0001-0005-0008-000000000001', 'complementary', 'wedding', 0.78, 78, 'Stunning jewelry and accessories'),

-- Decoration adjacencies
('22222222-0001-0001-0003-000000000001', '22222222-0001-0001-0009-000000000001', 'complementary', 'wedding', 0.88, 88, 'Fresh flowers elevate any decoration'),
('22222222-0001-0001-0003-000000000001', '22222222-0001-0001-0013-000000000001', 'complementary', 'wedding', 0.82, 82, 'Lighting enhances decorative elements'),
('22222222-0001-0001-0003-000000000001', '22222222-0001-0001-0018-000000000001', 'complementary', 'wedding', 0.75, 75, 'Quality furniture completes the look'),

-- Transport adjacencies
('22222222-0001-0001-0012-000000000001', '22222222-0001-0003-0001-000000000001', 'complementary', 'wedding', 0.70, 70, 'Arrange transportation for guests'),
('22222222-0001-0001-0012-000000000001', '22222222-0001-0003-0003-000000000001', 'complementary', 'wedding', 0.75, 75, 'Book accommodations for traveling guests'),

-- Fashion adjacencies for weddings
('22222222-0001-0001-0011-000000000001', '22222222-0001-0005-0002-000000000001', 'complementary', 'wedding', 0.90, 90, 'Custom tailoring for the perfect fit'),
('22222222-0001-0001-0011-000000000001', '22222222-0001-0005-0010-000000000001', 'complementary', 'wedding', 0.85, 85, 'Premium fabrics for your special outfit'),
('22222222-0001-0001-0011-000000000001', '22222222-0001-0005-0011-000000000001', 'complementary', 'wedding', 0.80, 80, 'Complete your look with custom shoes'),

-- Planner as hub
('22222222-0001-0001-0016-000000000001', '22222222-0001-0001-0001-000000000001', 'prerequisite', 'wedding', 0.95, 95, 'Planners help find the perfect venue'),
('22222222-0001-0001-0016-000000000001', '22222222-0001-0001-0002-000000000001', 'complementary', 'wedding', 0.92, 92, 'Planners coordinate with top caterers'),
('22222222-0001-0001-0016-000000000001', '22222222-0001-0001-0003-000000000001', 'complementary', 'wedding', 0.90, 90, 'Planners design cohesive themes');

-- HOME RENOVATION ADJACENCIES
INSERT INTO service_adjacencies (source_category_id, target_category_id, adjacency_type, trigger_context, base_affinity_score, cross_sell_priority, recommendation_copy) VALUES
-- Interior Design as hub
('22222222-0001-0002-0009-000000000001', '22222222-0001-0002-0005-000000000001', 'complementary', 'renovation', 0.90, 90, 'Fresh paint transforms spaces'),
('22222222-0001-0002-0009-000000000001', '22222222-0001-0002-0006-000000000001', 'complementary', 'renovation', 0.88, 88, 'Custom carpentry brings designs to life'),
('22222222-0001-0002-0009-000000000001', '22222222-0001-0002-0007-000000000001', 'complementary', 'renovation', 0.85, 85, 'Quality flooring completes the look'),
('22222222-0001-0002-0009-000000000001', '22222222-0001-0002-0015-000000000001', 'complementary', 'renovation', 0.80, 80, 'Window treatments add finishing touches'),
('22222222-0001-0002-0009-000000000001', '22222222-0001-0002-0008-000000000001', 'complementary', 'renovation', 0.75, 75, 'Climate control for comfort'),

-- Plumbing adjacencies
('22222222-0001-0002-0003-000000000001', '22222222-0001-0002-0007-000000000001', 'complementary', 'renovation', 0.85, 85, 'Tiling often follows plumbing work'),
('22222222-0001-0002-0003-000000000001', '22222222-0001-0002-0018-000000000001', 'complementary', 'renovation', 0.80, 80, 'Waterproofing prevents future issues'),
('22222222-0001-0002-0003-000000000001', '22222222-0001-0002-0004-000000000001', 'complementary', 'renovation', 0.70, 70, 'Coordinate with electrical work'),

-- Electrical adjacencies
('22222222-0001-0002-0004-000000000001', '22222222-0001-0002-0008-000000000001', 'complementary', 'renovation', 0.82, 82, 'AC requires proper electrical setup'),
('22222222-0001-0002-0004-000000000001', '22222222-0001-0002-0021-000000000001', 'complementary', 'renovation', 0.78, 78, 'Smart home needs professional wiring'),
('22222222-0001-0002-0004-000000000001', '22222222-0001-0002-0016-000000000001', 'complementary', 'renovation', 0.75, 75, 'Security systems need electrical work'),

-- Landscaping adjacencies
('22222222-0001-0002-0010-000000000001', '22222222-0001-0002-0017-000000000001', 'complementary', 'renovation', 0.75, 75, 'Pool complements outdoor spaces'),
('22222222-0001-0002-0010-000000000001', '22222222-0001-0002-0020-000000000001', 'complementary', 'renovation', 0.70, 70, 'Fencing and gates secure the property');

-- MOVING/RELOCATION ADJACENCIES
INSERT INTO service_adjacencies (source_category_id, target_category_id, adjacency_type, trigger_context, base_affinity_score, cross_sell_priority, recommendation_copy) VALUES
('22222222-0001-0002-0001-000000000001', '22222222-0001-0002-0002-000000000001', 'complementary', 'relocation', 0.92, 92, 'Deep clean your new home before moving in'),
('22222222-0001-0002-0001-000000000001', '22222222-0001-0002-0014-000000000001', 'complementary', 'relocation', 0.85, 85, 'Furniture assembly for your new space'),
('22222222-0001-0002-0001-000000000001', '22222222-0001-0002-0004-000000000001', 'complementary', 'relocation', 0.80, 80, 'Check electrical systems in new home'),
('22222222-0001-0002-0001-000000000001', '22222222-0001-0002-0003-000000000001', 'complementary', 'relocation', 0.78, 78, 'Verify plumbing is in order'),
('22222222-0001-0002-0001-000000000001', '22222222-0001-0002-0016-000000000001', 'complementary', 'relocation', 0.75, 75, 'Secure your new home'),
('22222222-0001-0002-0001-000000000001', '22222222-0001-0002-0011-000000000001', 'complementary', 'relocation', 0.72, 72, 'Pest control before unpacking'),
('22222222-0001-0002-0001-000000000001', '22222222-0001-0002-0008-000000000001', 'complementary', 'relocation', 0.70, 70, 'Service AC units in new home');

-- TRAVEL ADJACENCIES
INSERT INTO service_adjacencies (source_category_id, target_category_id, adjacency_type, trigger_context, base_affinity_score, cross_sell_priority, recommendation_copy) VALUES
-- Flight triggers
('22222222-0001-0003-0001-000000000001', '22222222-0001-0003-0004-000000000001', 'complementary', 'domestic_travel', 0.95, 95, 'Airport pickup and drop-off'),
('22222222-0001-0003-0001-000000000001', '22222222-0001-0003-0003-000000000001', 'complementary', 'domestic_travel', 0.90, 90, 'Book your accommodation'),
('22222222-0001-0003-0001-000000000001', '22222222-0001-0003-0002-000000000001', 'complementary', 'domestic_travel', 0.85, 85, 'Rent a car at destination'),

-- International travel
('22222222-0001-0003-0005-000000000001', '22222222-0001-0003-0008-000000000001', 'complementary', 'international_travel', 0.92, 92, 'Protect your trip with travel insurance'),
('22222222-0001-0003-0005-000000000001', '22222222-0001-0003-0007-000000000001', 'complementary', 'international_travel', 0.88, 88, 'Exchange currency before you travel'),
('22222222-0001-0003-0005-000000000001', '22222222-0001-0003-0009-000000000001', 'complementary', 'international_travel', 0.82, 82, 'Quality luggage for your journey'),
('22222222-0001-0003-0005-000000000001', '22222222-0001-0003-0010-000000000001', 'complementary', 'international_travel', 0.75, 75, 'Access airport lounges for comfort'),

-- Hotel to activities
('22222222-0001-0003-0003-000000000001', '22222222-0001-0003-0006-000000000001', 'complementary', 'vacation', 0.85, 85, 'Explore with local guides'),
('22222222-0001-0003-0003-000000000001', '22222222-0001-0003-0012-000000000001', 'complementary', 'vacation', 0.80, 80, 'Concierge services for convenience'),
('22222222-0001-0003-0003-000000000001', '22222222-0001-0003-0011-000000000001', 'complementary', 'business_travel', 0.82, 82, 'Chauffeur for business meetings');

-- AUTOMOTIVE ADJACENCIES
INSERT INTO service_adjacencies (source_category_id, target_category_id, adjacency_type, trigger_context, base_affinity_score, cross_sell_priority, recommendation_copy) VALUES
('22222222-0001-0009-0001-000000000001', '22222222-0001-0009-0006-000000000001', 'complementary', 'vehicle_purchase', 0.95, 95, 'Protect your investment with insurance'),
('22222222-0001-0009-0001-000000000001', '22222222-0001-0009-0008-000000000001', 'complementary', 'vehicle_purchase', 0.88, 88, 'Track and secure your vehicle'),
('22222222-0001-0009-0001-000000000001', '22222222-0001-0009-0009-000000000001', 'complementary', 'vehicle_purchase', 0.82, 82, 'Customize with tinting and accessories'),
('22222222-0001-0009-0001-000000000001', '22222222-0001-0009-0005-000000000001', 'complementary', 'vehicle_purchase', 0.75, 75, 'Keep your car pristine with detailing'),

-- Accident triggers
('22222222-0001-0009-0004-000000000001', '22222222-0001-0009-0003-000000000001', 'follow_up', 'vehicle_accident', 0.95, 95, 'Panel beating and body repair'),
('22222222-0001-0009-0004-000000000001', '22222222-0001-0009-0002-000000000001', 'follow_up', 'vehicle_accident', 0.90, 90, 'Mechanical repairs after accident'),
('22222222-0001-0009-0004-000000000001', '22222222-0001-0009-0007-000000000001', 'follow_up', 'vehicle_accident', 0.85, 85, 'Source replacement parts'),
('22222222-0001-0009-0003-000000000001', '22222222-0001-0009-0005-000000000001', 'follow_up', 'vehicle_accident', 0.80, 80, 'Detail and polish after repairs');

-- HEALTH ADJACENCIES
INSERT INTO service_adjacencies (source_category_id, target_category_id, adjacency_type, trigger_context, base_affinity_score, cross_sell_priority, recommendation_copy) VALUES
('22222222-0001-0008-0001-000000000001', '22222222-0001-0008-0002-000000000001', 'complementary', 'medical_recovery', 0.90, 90, 'Physiotherapy for faster recovery'),
('22222222-0001-0008-0001-000000000001', '22222222-0001-0008-0004-000000000001', 'complementary', 'medical_recovery', 0.85, 85, 'Medication delivered to your door'),
('22222222-0001-0008-0001-000000000001', '22222222-0001-0008-0007-000000000001', 'complementary', 'medical_recovery', 0.80, 80, 'Medical equipment for home care'),
('22222222-0001-0008-0001-000000000001', '22222222-0001-0008-0003-000000000001', 'complementary', 'elderly_care', 0.88, 88, 'Dedicated caregivers for daily support'),

-- Fitness journey
('22222222-0001-0005-0012-000000000001', '22222222-0001-0004-0006-000000000001', 'complementary', 'fitness', 0.88, 88, 'Nutrition guidance for results'),
('22222222-0001-0005-0012-000000000001', '22222222-0001-0004-0002-000000000001', 'complementary', 'fitness', 0.85, 85, 'Healthy meal prep services'),
('22222222-0001-0005-0012-000000000001', '22222222-0001-0008-0010-000000000001', 'complementary', 'fitness', 0.78, 78, 'Yoga for flexibility and mindfulness'),
('22222222-0001-0005-0012-000000000001', '22222222-0001-0005-0007-000000000001', 'complementary', 'fitness', 0.72, 72, 'Spa recovery for sore muscles'),

-- Childbirth
('22222222-0001-0008-0009-000000000001', '22222222-0001-0008-0001-000000000001', 'complementary', 'childbirth', 0.85, 85, 'Post-natal nursing care'),
('22222222-0001-0008-0009-000000000001', '22222222-0001-0004-0002-000000000001', 'complementary', 'childbirth', 0.80, 80, 'Nutritious meals for new mothers');

-- BUSINESS ADJACENCIES
INSERT INTO service_adjacencies (source_category_id, target_category_id, adjacency_type, trigger_context, base_affinity_score, cross_sell_priority, recommendation_copy) VALUES
('22222222-0001-0006-0001-000000000001', '22222222-0001-0006-0002-000000000001', 'prerequisite', 'business_launch', 0.92, 92, 'Legal setup protects your business'),
('22222222-0001-0006-0001-000000000001', '22222222-0001-0006-0003-000000000001', 'complementary', 'business_launch', 0.90, 90, 'Proper accounting from day one'),
('22222222-0001-0006-0001-000000000001', '22222222-0001-0006-0005-000000000001', 'complementary', 'business_launch', 0.88, 88, 'Create a memorable brand identity'),
('22222222-0001-0006-0001-000000000001', '22222222-0001-0006-0006-000000000001', 'complementary', 'business_launch', 0.85, 85, 'Build your online presence'),
('22222222-0001-0006-0001-000000000001', '22222222-0001-0006-0011-000000000001', 'complementary', 'business_launch', 0.82, 82, 'Protect your business with insurance'),

-- Office setup
('22222222-0001-0006-0009-000000000001', '22222222-0001-0006-0008-000000000001', 'complementary', 'office_setup', 0.88, 88, 'IT infrastructure and support'),
('22222222-0001-0006-0009-000000000001', '22222222-0001-0002-0002-000000000001', 'complementary', 'office_setup', 0.82, 82, 'Office cleaning services'),
('22222222-0001-0006-0009-000000000001', '22222222-0001-0002-0016-000000000001', 'complementary', 'office_setup', 0.78, 78, 'Security systems for your office');

-- PROPERTY DEVELOPMENT ADJACENCIES
INSERT INTO service_adjacencies (source_category_id, target_category_id, adjacency_type, trigger_context, base_affinity_score, cross_sell_priority, recommendation_copy) VALUES
('22222222-0001-0011-0001-000000000001', '22222222-0001-0011-0006-000000000001', 'prerequisite', 'property_development', 0.95, 95, 'Structural engineering ensures safety'),
('22222222-0001-0011-0001-000000000001', '22222222-0001-0011-0005-000000000001', 'complementary', 'property_development', 0.92, 92, 'Accurate cost estimation'),
('22222222-0001-0011-0001-000000000001', '22222222-0001-0011-0003-000000000001', 'complementary', 'property_development', 0.90, 90, 'Trusted contractors bring designs to life'),
('22222222-0001-0011-0001-000000000001', '22222222-0001-0011-0002-000000000001', 'prerequisite', 'property_development', 0.88, 88, 'Land survey before construction'),

-- Property sale
('22222222-0001-0011-0004-000000000001', '22222222-0001-0010-0001-000000000001', 'complementary', 'property_sale', 0.85, 85, 'Professional photos sell properties'),
('22222222-0001-0011-0004-000000000001', '22222222-0001-0011-0009-000000000001', 'complementary', 'property_sale', 0.82, 82, 'Accurate valuation for best price'),
('22222222-0001-0011-0004-000000000001', '22222222-0001-0011-0010-000000000001', 'complementary', 'property_sale', 0.80, 80, 'Legal conveyancing for smooth sale');

-- ENERGY ADJACENCIES
INSERT INTO service_adjacencies (source_category_id, target_category_id, adjacency_type, trigger_context, base_affinity_score, cross_sell_priority, recommendation_copy) VALUES
('22222222-0001-0012-0001-000000000001', '22222222-0001-0012-0003-000000000001', 'complementary', 'solar_installation', 0.92, 92, 'Battery storage for nighttime power'),
('22222222-0001-0012-0001-000000000001', '22222222-0001-0012-0005-000000000001', 'prerequisite', 'solar_installation', 0.88, 88, 'Energy audit optimizes your system'),
('22222222-0001-0012-0001-000000000001', '22222222-0001-0002-0004-000000000001', 'complementary', 'solar_installation', 0.85, 85, 'Electrical work for solar integration'),

('22222222-0001-0012-0002-000000000001', '22222222-0001-0012-0004-000000000001', 'complementary', 'generator_setup', 0.90, 90, 'Reliable fuel delivery service'),
('22222222-0001-0012-0002-000000000001', '22222222-0001-0002-0004-000000000001', 'complementary', 'generator_setup', 0.88, 88, 'Proper electrical installation');

-- CREATIVE/CONTENT ADJACENCIES
INSERT INTO service_adjacencies (source_category_id, target_category_id, adjacency_type, trigger_context, base_affinity_score, cross_sell_priority, recommendation_copy) VALUES
('22222222-0001-0010-0001-000000000001', '22222222-0001-0010-0002-000000000001', 'complementary', 'content_production', 0.88, 88, 'Video complements photo content'),
('22222222-0001-0010-0001-000000000001', '22222222-0001-0005-0001-000000000001', 'complementary', 'content_production', 0.85, 85, 'Stylists elevate photo shoots'),
('22222222-0001-0010-0001-000000000001', '22222222-0001-0001-0010-000000000001', 'complementary', 'content_production', 0.82, 82, 'Professional makeup for shoots'),

('22222222-0001-0010-0002-000000000001', '22222222-0001-0010-0010-000000000001', 'follow_up', 'content_production', 0.90, 90, 'Post-production editing'),
('22222222-0001-0010-0002-000000000001', '22222222-0001-0010-0005-000000000001', 'complementary', 'content_production', 0.85, 85, 'Voice over for your videos'),
('22222222-0001-0010-0002-000000000001', '22222222-0001-0010-0009-000000000001', 'complementary', 'content_production', 0.80, 80, 'Animation for dynamic content'),

('22222222-0001-0010-0003-000000000001', '22222222-0001-0010-0004-000000000001', 'complementary', 'branding', 0.88, 88, 'Copy that matches your visuals'),
('22222222-0001-0010-0003-000000000001', '22222222-0001-0010-0007-000000000001', 'complementary', 'branding', 0.85, 85, 'Social media management'),
('22222222-0001-0010-0003-000000000001', '22222222-0001-0006-0006-000000000001', 'complementary', 'branding', 0.82, 82, 'Website to showcase your brand');

-- ============================================================================
-- SECTION 4: EVENT-CATEGORY MAPPINGS
-- ============================================================================

-- Wedding event mappings
INSERT INTO event_category_mappings (event_trigger_id, category_id, role_type, phase, typical_booking_offset_days, necessity_score, popularity_score, typical_budget_percentage) VALUES
('11111111-0001-0001-0001-000000000001', '22222222-0001-0001-0001-000000000001', 'primary', 'planning', 180, 0.95, 0.98, 25),
('11111111-0001-0001-0001-000000000001', '22222222-0001-0001-0002-000000000001', 'primary', 'planning', 120, 0.92, 0.95, 20),
('11111111-0001-0001-0001-000000000001', '22222222-0001-0001-0003-000000000001', 'primary', 'planning', 90, 0.88, 0.92, 10),
('11111111-0001-0001-0001-000000000001', '22222222-0001-0001-0004-000000000001', 'primary', 'planning', 90, 0.90, 0.95, 8),
('11111111-0001-0001-0001-000000000001', '22222222-0001-0001-0005-000000000001', 'secondary', 'planning', 90, 0.75, 0.80, 6),
('11111111-0001-0001-0001-000000000001', '22222222-0001-0001-0006-000000000001', 'primary', 'planning', 60, 0.85, 0.90, 5),
('11111111-0001-0001-0001-000000000001', '22222222-0001-0001-0008-000000000001', 'primary', 'pre_event', 30, 0.90, 0.95, 3),
('11111111-0001-0001-0001-000000000001', '22222222-0001-0001-0010-000000000001', 'primary', 'pre_event', 7, 0.92, 0.98, 5),
('11111111-0001-0001-0001-000000000001', '22222222-0001-0005-0002-000000000001', 'primary', 'planning', 60, 0.88, 0.92, 8),
('11111111-0001-0001-0001-000000000001', '22222222-0001-0001-0016-000000000001', 'optional', 'planning', 180, 0.40, 0.50, 5);

-- Relocation event mappings
INSERT INTO event_category_mappings (event_trigger_id, category_id, role_type, phase, typical_booking_offset_days, necessity_score, popularity_score, typical_budget_percentage) VALUES
('11111111-0001-0002-0001-000000000002', '22222222-0001-0002-0001-000000000001', 'primary', 'event_day', 7, 0.95, 0.98, 40),
('11111111-0001-0002-0001-000000000002', '22222222-0001-0002-0002-000000000001', 'primary', 'pre_event', 3, 0.88, 0.92, 15),
('11111111-0001-0002-0001-000000000002', '22222222-0001-0002-0004-000000000001', 'secondary', 'post_event', 1, 0.70, 0.75, 10),
('11111111-0001-0002-0001-000000000002', '22222222-0001-0002-0003-000000000001', 'secondary', 'post_event', 1, 0.65, 0.70, 8),
('11111111-0001-0002-0001-000000000002', '22222222-0001-0002-0016-000000000001', 'secondary', 'post_event', 3, 0.60, 0.65, 12);

-- International travel mappings
INSERT INTO event_category_mappings (event_trigger_id, category_id, role_type, phase, typical_booking_offset_days, necessity_score, popularity_score, typical_budget_percentage) VALUES
('11111111-0001-0003-0001-000000000002', '22222222-0001-0003-0005-000000000001', 'primary', 'planning', 60, 0.95, 0.98, 10),
('11111111-0001-0003-0001-000000000002', '22222222-0001-0003-0008-000000000001', 'primary', 'planning', 30, 0.85, 0.88, 5),
('11111111-0001-0003-0001-000000000002', '22222222-0001-0003-0007-000000000001', 'primary', 'pre_event', 7, 0.80, 0.85, 15),
('11111111-0001-0003-0001-000000000002', '22222222-0001-0003-0001-000000000001', 'primary', 'event_day', 1, 0.90, 0.95, 8),
('11111111-0001-0003-0001-000000000002', '22222222-0001-0003-0003-000000000001', 'primary', 'planning', 30, 0.92, 0.95, 40);

-- ============================================================================
-- END OF SEED DATA
-- ============================================================================
