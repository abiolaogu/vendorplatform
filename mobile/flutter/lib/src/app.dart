// =============================================================================
// VENDORPLATFORM APP
// Root app widget with theme, routing, and providers
// =============================================================================

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import 'core/theme/app_theme.dart';
import 'core/routing/app_router.dart';
import 'features/auth/providers/auth_provider.dart';

class VendorPlatformApp extends ConsumerWidget {
  const VendorPlatformApp({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final router = ref.watch(routerProvider);

    return MaterialApp.router(
      title: 'VendorPlatform',
      debugShowCheckedModeBanner: false,
      theme: AppTheme.lightTheme,
      darkTheme: AppTheme.darkTheme,
      themeMode: ThemeMode.system,
      routerConfig: router,
    );
  }
}

// =============================================================================
// ROUTER PROVIDER
// =============================================================================

final routerProvider = Provider<GoRouter>((ref) {
  final authState = ref.watch(authStateProvider);

  return GoRouter(
    initialLocation: '/',
    debugLogDiagnostics: true,
    redirect: (context, state) {
      final isLoggedIn = authState.valueOrNull?.isLoggedIn ?? false;
      final isLoggingIn = state.matchedLocation == '/login' ||
          state.matchedLocation == '/register';
      final isOnboarding = state.matchedLocation == '/onboarding';

      // If not logged in and not on auth pages, redirect to login
      if (!isLoggedIn && !isLoggingIn && !isOnboarding) {
        return '/login';
      }

      // If logged in and on auth pages, redirect to home
      if (isLoggedIn && isLoggingIn) {
        return '/';
      }

      return null;
    },
    routes: appRoutes,
  );
});

// =============================================================================
// APP ROUTES
// =============================================================================

final appRoutes = <RouteBase>[
  // Splash
  GoRoute(
    path: '/splash',
    builder: (context, state) => const SplashScreen(),
  ),

  // Onboarding
  GoRoute(
    path: '/onboarding',
    builder: (context, state) => const OnboardingScreen(),
  ),

  // Authentication
  GoRoute(
    path: '/login',
    builder: (context, state) => const LoginScreen(),
  ),
  GoRoute(
    path: '/register',
    builder: (context, state) => const RegisterScreen(),
  ),
  GoRoute(
    path: '/forgot-password',
    builder: (context, state) => const ForgotPasswordScreen(),
  ),

  // Main Shell
  ShellRoute(
    builder: (context, state, child) => MainShell(child: child),
    routes: [
      // Home
      GoRoute(
        path: '/',
        builder: (context, state) => const HomeScreen(),
      ),

      // Search
      GoRoute(
        path: '/search',
        builder: (context, state) => const SearchScreen(),
      ),

      // Bookings
      GoRoute(
        path: '/bookings',
        builder: (context, state) => const BookingsScreen(),
        routes: [
          GoRoute(
            path: ':id',
            builder: (context, state) {
              final id = state.pathParameters['id']!;
              return BookingDetailScreen(bookingId: id);
            },
          ),
        ],
      ),

      // Messages
      GoRoute(
        path: '/messages',
        builder: (context, state) => const MessagesScreen(),
        routes: [
          GoRoute(
            path: ':id',
            builder: (context, state) {
              final id = state.pathParameters['id']!;
              return ChatScreen(conversationId: id);
            },
          ),
        ],
      ),

      // Profile
      GoRoute(
        path: '/profile',
        builder: (context, state) => const ProfileScreen(),
        routes: [
          GoRoute(
            path: 'edit',
            builder: (context, state) => const EditProfileScreen(),
          ),
          GoRoute(
            path: 'settings',
            builder: (context, state) => const SettingsScreen(),
          ),
          GoRoute(
            path: 'notifications',
            builder: (context, state) => const NotificationSettingsScreen(),
          ),
        ],
      ),
    ],
  ),

  // Vendor Details
  GoRoute(
    path: '/vendor/:id',
    builder: (context, state) {
      final id = state.pathParameters['id']!;
      return VendorDetailScreen(vendorId: id);
    },
  ),

  // Service Booking Flow
  GoRoute(
    path: '/book/:serviceId',
    builder: (context, state) {
      final serviceId = state.pathParameters['serviceId']!;
      return BookingFlowScreen(serviceId: serviceId);
    },
  ),

  // Emergency (HomeRescue)
  GoRoute(
    path: '/emergency',
    builder: (context, state) => const EmergencyScreen(),
    routes: [
      GoRoute(
        path: 'new',
        builder: (context, state) => const NewEmergencyScreen(),
      ),
      GoRoute(
        path: ':id',
        builder: (context, state) {
          final id = state.pathParameters['id']!;
          return EmergencyTrackingScreen(requestId: id);
        },
      ),
    ],
  ),

  // Life Events (LifeOS)
  GoRoute(
    path: '/events',
    builder: (context, state) => const LifeEventsScreen(),
    routes: [
      GoRoute(
        path: ':id',
        builder: (context, state) {
          final id = state.pathParameters['id']!;
          return EventOrchestratorScreen(eventId: id);
        },
      ),
    ],
  ),

  // AI Assistant (EventGPT)
  GoRoute(
    path: '/assistant',
    builder: (context, state) => const AIAssistantScreen(),
  ),
];

// =============================================================================
// PLACEHOLDER SCREENS
// TODO: Implement actual screens
// =============================================================================

class SplashScreen extends StatelessWidget {
  const SplashScreen({super.key});
  @override
  Widget build(BuildContext context) => const Scaffold(body: Center(child: CircularProgressIndicator()));
}

class OnboardingScreen extends StatelessWidget {
  const OnboardingScreen({super.key});
  @override
  Widget build(BuildContext context) => const Scaffold(body: Center(child: Text('Onboarding')));
}

class LoginScreen extends StatelessWidget {
  const LoginScreen({super.key});
  @override
  Widget build(BuildContext context) => const Scaffold(body: Center(child: Text('Login')));
}

class RegisterScreen extends StatelessWidget {
  const RegisterScreen({super.key});
  @override
  Widget build(BuildContext context) => const Scaffold(body: Center(child: Text('Register')));
}

class ForgotPasswordScreen extends StatelessWidget {
  const ForgotPasswordScreen({super.key});
  @override
  Widget build(BuildContext context) => const Scaffold(body: Center(child: Text('Forgot Password')));
}

class MainShell extends StatelessWidget {
  final Widget child;
  const MainShell({super.key, required this.child});
  @override
  Widget build(BuildContext context) => Scaffold(
    body: child,
    bottomNavigationBar: const BottomNavBar(),
  );
}

class BottomNavBar extends StatelessWidget {
  const BottomNavBar({super.key});
  @override
  Widget build(BuildContext context) => BottomNavigationBar(
    type: BottomNavigationBarType.fixed,
    items: const [
      BottomNavigationBarItem(icon: Icon(Icons.home), label: 'Home'),
      BottomNavigationBarItem(icon: Icon(Icons.search), label: 'Search'),
      BottomNavigationBarItem(icon: Icon(Icons.calendar_today), label: 'Bookings'),
      BottomNavigationBarItem(icon: Icon(Icons.message), label: 'Messages'),
      BottomNavigationBarItem(icon: Icon(Icons.person), label: 'Profile'),
    ],
  );
}

class HomeScreen extends StatelessWidget {
  const HomeScreen({super.key});
  @override
  Widget build(BuildContext context) => const Center(child: Text('Home'));
}

class SearchScreen extends StatelessWidget {
  const SearchScreen({super.key});
  @override
  Widget build(BuildContext context) => const Center(child: Text('Search'));
}

class BookingsScreen extends StatelessWidget {
  const BookingsScreen({super.key});
  @override
  Widget build(BuildContext context) => const Center(child: Text('Bookings'));
}

class BookingDetailScreen extends StatelessWidget {
  final String bookingId;
  const BookingDetailScreen({super.key, required this.bookingId});
  @override
  Widget build(BuildContext context) => Center(child: Text('Booking: $bookingId'));
}

class MessagesScreen extends StatelessWidget {
  const MessagesScreen({super.key});
  @override
  Widget build(BuildContext context) => const Center(child: Text('Messages'));
}

class ChatScreen extends StatelessWidget {
  final String conversationId;
  const ChatScreen({super.key, required this.conversationId});
  @override
  Widget build(BuildContext context) => Center(child: Text('Chat: $conversationId'));
}

class ProfileScreen extends StatelessWidget {
  const ProfileScreen({super.key});
  @override
  Widget build(BuildContext context) => const Center(child: Text('Profile'));
}

class EditProfileScreen extends StatelessWidget {
  const EditProfileScreen({super.key});
  @override
  Widget build(BuildContext context) => const Center(child: Text('Edit Profile'));
}

class SettingsScreen extends StatelessWidget {
  const SettingsScreen({super.key});
  @override
  Widget build(BuildContext context) => const Center(child: Text('Settings'));
}

class NotificationSettingsScreen extends StatelessWidget {
  const NotificationSettingsScreen({super.key});
  @override
  Widget build(BuildContext context) => const Center(child: Text('Notification Settings'));
}

class VendorDetailScreen extends StatelessWidget {
  final String vendorId;
  const VendorDetailScreen({super.key, required this.vendorId});
  @override
  Widget build(BuildContext context) => Center(child: Text('Vendor: $vendorId'));
}

class BookingFlowScreen extends StatelessWidget {
  final String serviceId;
  const BookingFlowScreen({super.key, required this.serviceId});
  @override
  Widget build(BuildContext context) => Center(child: Text('Book Service: $serviceId'));
}

class EmergencyScreen extends StatelessWidget {
  const EmergencyScreen({super.key});
  @override
  Widget build(BuildContext context) => const Center(child: Text('Emergency Services'));
}

class NewEmergencyScreen extends StatelessWidget {
  const NewEmergencyScreen({super.key});
  @override
  Widget build(BuildContext context) => const Center(child: Text('New Emergency'));
}

class EmergencyTrackingScreen extends StatelessWidget {
  final String requestId;
  const EmergencyTrackingScreen({super.key, required this.requestId});
  @override
  Widget build(BuildContext context) => Center(child: Text('Track Emergency: $requestId'));
}

class LifeEventsScreen extends StatelessWidget {
  const LifeEventsScreen({super.key});
  @override
  Widget build(BuildContext context) => const Center(child: Text('Life Events'));
}

class EventOrchestratorScreen extends StatelessWidget {
  final String eventId;
  const EventOrchestratorScreen({super.key, required this.eventId});
  @override
  Widget build(BuildContext context) => Center(child: Text('Event: $eventId'));
}

class AIAssistantScreen extends StatelessWidget {
  const AIAssistantScreen({super.key});
  @override
  Widget build(BuildContext context) => const Center(child: Text('AI Assistant'));
}
