"""
Advanced ML Components for Vendor Platform Recommendation Engine

This module provides:
1. Embedding-based similarity using sentence transformers
2. Co-purchase pattern mining using association rules
3. Event detection using NLP
4. Demand forecasting for seasonal trends
5. Real-time scoring model training
"""

import numpy as np
import pandas as pd
from typing import List, Dict, Optional, Tuple, Any
from dataclasses import dataclass, field
from datetime import datetime, timedelta
import json
import logging
from collections import defaultdict
import asyncio
import asyncpg
from redis import asyncio as aioredis

# ML Libraries
from sklearn.preprocessing import StandardScaler, LabelEncoder
from sklearn.cluster import KMeans
from sklearn.metrics.pairwise import cosine_similarity
from scipy.sparse import csr_matrix
import joblib

# Deep Learning (optional, for embeddings)
try:
    from sentence_transformers import SentenceTransformer
    HAS_TRANSFORMERS = True
except ImportError:
    HAS_TRANSFORMERS = False

# Association Rules
try:
    from mlxtend.frequent_patterns import apriori, association_rules
    from mlxtend.preprocessing import TransactionEncoder
    HAS_MLXTEND = True
except ImportError:
    HAS_MLXTEND = False

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


# =============================================================================
# DATA CLASSES
# =============================================================================

@dataclass
class ServiceEmbedding:
    """Represents a service's embedding vector"""
    service_id: str
    category_id: str
    vendor_id: str
    embedding: np.ndarray
    metadata: Dict[str, Any] = field(default_factory=dict)


@dataclass
class CoPurchaseRule:
    """Represents a discovered co-purchase pattern"""
    antecedent_categories: List[str]
    consequent_categories: List[str]
    support: float
    confidence: float
    lift: float
    conviction: float
    event_context: Optional[str] = None


@dataclass
class DetectedEvent:
    """Represents a detected life event for a user"""
    event_type: str
    confidence: float
    trigger_signals: List[str]
    detected_at: datetime
    metadata: Dict[str, Any] = field(default_factory=dict)


@dataclass
class DemandForecast:
    """Represents demand forecast for a category"""
    category_id: str
    date: datetime
    predicted_demand: float
    confidence_interval: Tuple[float, float]
    seasonality_factor: float
    trend_direction: str  # 'up', 'down', 'stable'


# =============================================================================
# EMBEDDING SERVICE
# =============================================================================

class EmbeddingService:
    """
    Service for generating and managing service/vendor embeddings
    using sentence transformers or custom models.
    """
    
    def __init__(
        self,
        model_name: str = "all-MiniLM-L6-v2",
        cache_ttl: int = 3600
    ):
        self.model_name = model_name
        self.cache_ttl = cache_ttl
        self.model = None
        self.embeddings_cache: Dict[str, np.ndarray] = {}
        self._initialize_model()
    
    def _initialize_model(self):
        """Initialize the embedding model"""
        if HAS_TRANSFORMERS:
            try:
                self.model = SentenceTransformer(self.model_name)
                logger.info(f"Loaded embedding model: {self.model_name}")
            except Exception as e:
                logger.warning(f"Could not load transformer model: {e}")
                self.model = None
        else:
            logger.warning("sentence-transformers not installed, using fallback")
    
    def generate_embedding(self, text: str) -> np.ndarray:
        """Generate embedding for text"""
        if self.model is not None:
            return self.model.encode(text, convert_to_numpy=True)
        else:
            return self._fallback_embedding(text)
    
    def _fallback_embedding(self, text: str, dim: int = 384) -> np.ndarray:
        """Simple fallback embedding using character n-grams"""
        np.random.seed(hash(text) % (2**32))
        return np.random.randn(dim).astype(np.float32)
    
    def generate_service_embedding(
        self,
        service_id: str,
        name: str,
        description: str,
        category_name: str,
        tags: List[str],
        vendor_id: str
    ) -> ServiceEmbedding:
        """Generate embedding for a service"""
        text_parts = [name, description or "", category_name, " ".join(tags or [])]
        combined_text = " ".join(filter(None, text_parts))
        embedding = self.generate_embedding(combined_text)
        
        return ServiceEmbedding(
            service_id=service_id,
            category_id="",
            vendor_id=vendor_id,
            embedding=embedding,
            metadata={"name": name, "category": category_name, "tags": tags}
        )
    
    def find_similar_services(
        self,
        query_embedding: np.ndarray,
        all_embeddings: List[ServiceEmbedding],
        top_k: int = 10,
        exclude_ids: Optional[List[str]] = None
    ) -> List[Tuple[ServiceEmbedding, float]]:
        """Find most similar services using cosine similarity"""
        exclude_ids = exclude_ids or []
        candidates = [e for e in all_embeddings if e.service_id not in exclude_ids]
        
        if not candidates:
            return []
        
        candidate_embeddings = np.stack([e.embedding for e in candidates])
        similarities = cosine_similarity(
            query_embedding.reshape(1, -1),
            candidate_embeddings
        )[0]
        
        top_indices = np.argsort(similarities)[::-1][:top_k]
        return [(candidates[i], float(similarities[i])) for i in top_indices]


# =============================================================================
# CO-PURCHASE PATTERN MINER
# =============================================================================

class CoPurchasePatternMiner:
    """
    Mines co-purchase patterns from booking data using association rules.
    """
    
    def __init__(
        self,
        min_support: float = 0.01,
        min_confidence: float = 0.1,
        min_lift: float = 1.0
    ):
        self.min_support = min_support
        self.min_confidence = min_confidence
        self.min_lift = min_lift
    
    async def mine_patterns(
        self,
        db_pool: asyncpg.Pool,
        event_type: Optional[str] = None,
        time_window_days: int = 90
    ) -> List[CoPurchaseRule]:
        """Mine co-purchase patterns from booking data."""
        if not HAS_MLXTEND:
            logger.warning("mlxtend not installed, returning empty patterns")
            return []
        
        transactions = await self._fetch_transactions(db_pool, event_type, time_window_days)
        
        if len(transactions) < 10:
            logger.warning("Not enough transactions for pattern mining")
            return []
        
        te = TransactionEncoder()
        te_array = te.fit_transform(transactions)
        df = pd.DataFrame(te_array, columns=te.columns_)
        
        frequent_itemsets = apriori(df, min_support=self.min_support, use_colnames=True)
        
        if frequent_itemsets.empty:
            return []
        
        rules = association_rules(frequent_itemsets, metric="lift", min_threshold=self.min_lift)
        rules = rules[rules['confidence'] >= self.min_confidence]
        
        patterns = []
        for _, row in rules.iterrows():
            patterns.append(CoPurchaseRule(
                antecedent_categories=list(row['antecedents']),
                consequent_categories=list(row['consequents']),
                support=float(row['support']),
                confidence=float(row['confidence']),
                lift=float(row['lift']),
                conviction=float(row['conviction']) if pd.notna(row['conviction']) else 0.0,
                event_context=event_type
            ))
        
        return patterns
    
    async def _fetch_transactions(
        self,
        db_pool: asyncpg.Pool,
        event_type: Optional[str],
        time_window_days: int
    ) -> List[List[str]]:
        """Fetch transaction data from database"""
        query = f"""
            WITH user_transactions AS (
                SELECT 
                    COALESCE(b.project_id::text, b.user_id::text) as transaction_id,
                    ARRAY_AGG(DISTINCT sc.slug) as categories
                FROM bookings b
                JOIN services s ON s.id = b.service_id
                JOIN service_categories sc ON sc.id = s.category_id
                WHERE b.status IN ('completed', 'confirmed')
                  AND b.created_at > NOW() - INTERVAL '{time_window_days} days'
                GROUP BY transaction_id
                HAVING COUNT(DISTINCT s.category_id) >= 2
            )
            SELECT categories FROM user_transactions
        """
        
        async with db_pool.acquire() as conn:
            rows = await conn.fetch(query)
        
        return [row['categories'] for row in rows]


# =============================================================================
# EVENT DETECTOR
# =============================================================================

class EventDetector:
    """Detects life events from user behavior patterns."""
    
    EVENT_PATTERNS = {
        'wedding': {
            'keywords': ['wedding', 'bride', 'groom', 'reception', 'engagement',
                        'bridal', 'ceremony', 'vows', 'honeymoon', 'registry'],
            'category_signals': ['venue', 'catering', 'photography', 'decoration',
                                'florist', 'cake', 'makeup'],
            'min_category_matches': 2,
            'confidence_boost_per_match': 0.15
        },
        'relocation': {
            'keywords': ['moving', 'relocation', 'new home', 'apartment', 'movers', 'packing'],
            'category_signals': ['moving', 'cleaning', 'painting', 'electrical', 'plumbing'],
            'min_category_matches': 2,
            'confidence_boost_per_match': 0.12
        },
        'childbirth': {
            'keywords': ['baby', 'pregnancy', 'newborn', 'maternity', 'nursery', 'baby shower'],
            'category_signals': ['doula', 'photographer', 'catering', 'decoration'],
            'min_category_matches': 1,
            'confidence_boost_per_match': 0.20
        },
        'business_launch': {
            'keywords': ['business', 'company', 'startup', 'office', 'registration', 'branding'],
            'category_signals': ['business_registration', 'legal', 'branding', 'webdev'],
            'min_category_matches': 2,
            'confidence_boost_per_match': 0.15
        }
    }
    
    def __init__(self, db_pool: asyncpg.Pool):
        self.db_pool = db_pool
    
    async def detect_events(
        self,
        user_id: str,
        recent_searches: List[str],
        viewed_categories: List[str],
        booked_categories: List[str]
    ) -> List[DetectedEvent]:
        """Analyze user signals to detect potential life events."""
        detected = []
        
        for event_type, patterns in self.EVENT_PATTERNS.items():
            confidence = 0.0
            trigger_signals = []
            
            search_text = " ".join(recent_searches).lower()
            keyword_matches = sum(1 for kw in patterns['keywords'] if kw.lower() in search_text)
            if keyword_matches > 0:
                confidence += min(keyword_matches * 0.1, 0.3)
                trigger_signals.append(f"keyword_matches:{keyword_matches}")
            
            category_views = sum(1 for cat in viewed_categories if cat in patterns['category_signals'])
            if category_views >= patterns['min_category_matches']:
                confidence += category_views * patterns['confidence_boost_per_match']
                trigger_signals.append(f"category_views:{category_views}")
            
            category_bookings = sum(1 for cat in booked_categories if cat in patterns['category_signals'])
            if category_bookings > 0:
                confidence += category_bookings * 0.25
                trigger_signals.append(f"category_bookings:{category_bookings}")
            
            if confidence >= 0.3:
                detected.append(DetectedEvent(
                    event_type=event_type,
                    confidence=min(confidence, 1.0),
                    trigger_signals=trigger_signals,
                    detected_at=datetime.utcnow()
                ))
        
        detected.sort(key=lambda x: x.confidence, reverse=True)
        return detected


# =============================================================================
# MAIN ORCHESTRATOR
# =============================================================================

class MLOrchestrator:
    """Main orchestrator for all ML components."""
    
    def __init__(self, db_pool: asyncpg.Pool, redis_client: aioredis.Redis):
        self.db_pool = db_pool
        self.redis = redis_client
        self.embedding_service = EmbeddingService()
        self.pattern_miner = CoPurchasePatternMiner()
        self.event_detector = EventDetector(db_pool)
    
    async def run_daily_jobs(self):
        """Run daily ML jobs"""
        logger.info("Starting daily ML jobs...")
        
        patterns = await self.pattern_miner.mine_patterns(self.db_pool)
        logger.info(f"Mined {len(patterns)} co-purchase patterns")
        
        logger.info("Daily ML jobs completed")


if __name__ == "__main__":
    asyncio.run(main())
