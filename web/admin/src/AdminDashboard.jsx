// =============================================================================
// VENDORPLATFORM ADMIN DASHBOARD
// React-based admin interface for platform management
// =============================================================================

import React, { useState, useEffect } from 'react';
import {
  Box,
  CssBaseline,
  ThemeProvider,
  createTheme,
  AppBar,
  Toolbar,
  Typography,
  Drawer,
  List,
  ListItem,
  ListItemIcon,
  ListItemText,
  IconButton,
  Badge,
  Avatar,
  Divider,
  Card,
  CardContent,
  Grid,
  Button,
} from '@mui/material';
import {
  Dashboard,
  People,
  Store,
  ShoppingCart,
  Assessment,
  Settings,
  Notifications,
  Menu,
  Home,
  Warning,
  AttachMoney,
  Message,
  TrendingUp,
  Groups,
} from '@mui/icons-material';

// =============================================================================
// THEME
// =============================================================================

const theme = createTheme({
  palette: {
    primary: {
      main: '#2563EB',
      light: '#60A5FA',
      dark: '#1D4ED8',
    },
    secondary: {
      main: '#10B981',
      light: '#34D399',
      dark: '#059669',
    },
    error: {
      main: '#EF4444',
    },
    warning: {
      main: '#F59E0B',
    },
    success: {
      main: '#22C55E',
    },
    background: {
      default: '#F9FAFB',
      paper: '#FFFFFF',
    },
  },
  typography: {
    fontFamily: '"Inter", "Roboto", "Helvetica", "Arial", sans-serif',
  },
  shape: {
    borderRadius: 12,
  },
  components: {
    MuiCard: {
      styleOverrides: {
        root: {
          boxShadow: '0 1px 3px rgba(0,0,0,0.1)',
        },
      },
    },
    MuiButton: {
      styleOverrides: {
        root: {
          textTransform: 'none',
          fontWeight: 600,
        },
      },
    },
  },
});

const drawerWidth = 260;

// =============================================================================
// NAVIGATION ITEMS
// =============================================================================

const navigationItems = [
  { text: 'Dashboard', icon: <Dashboard />, path: '/admin' },
  { text: 'Users', icon: <People />, path: '/admin/users' },
  { text: 'Vendors', icon: <Store />, path: '/admin/vendors' },
  { text: 'Bookings', icon: <ShoppingCart />, path: '/admin/bookings' },
  { text: 'Transactions', icon: <AttachMoney />, path: '/admin/transactions' },
  { text: 'Analytics', icon: <Assessment />, path: '/admin/analytics' },
  { divider: true },
  { text: 'HomeRescue', icon: <Home />, path: '/admin/homerescue' },
  { text: 'Emergency Queue', icon: <Warning />, path: '/admin/emergencies' },
  { text: 'VendorNet', icon: <Groups />, path: '/admin/vendornet' },
  { divider: true },
  { text: 'Messages', icon: <Message />, path: '/admin/messages' },
  { text: 'Notifications', icon: <Notifications />, path: '/admin/notifications' },
  { text: 'Settings', icon: <Settings />, path: '/admin/settings' },
];

// =============================================================================
// DASHBOARD STATS
// =============================================================================

const DashboardStats = () => {
  const stats = [
    { title: 'Total Users', value: '12,456', change: '+12%', icon: <People />, color: '#2563EB' },
    { title: 'Active Vendors', value: '2,345', change: '+8%', icon: <Store />, color: '#10B981' },
    { title: 'Bookings Today', value: '456', change: '+23%', icon: <ShoppingCart />, color: '#F59E0B' },
    { title: 'Revenue (MTD)', value: '₦45.6M', change: '+18%', icon: <AttachMoney />, color: '#8B5CF6' },
    { title: 'Active Emergencies', value: '12', change: '-5%', icon: <Warning />, color: '#EF4444' },
    { title: 'Partnerships', value: '234', change: '+15%', icon: <Groups />, color: '#06B6D4' },
  ];

  return (
    <Grid container spacing={3}>
      {stats.map((stat, index) => (
        <Grid item xs={12} sm={6} md={4} lg={2} key={index}>
          <Card>
            <CardContent>
              <Box sx={{ display: 'flex', alignItems: 'center', mb: 2 }}>
                <Box
                  sx={{
                    backgroundColor: `${stat.color}20`,
                    borderRadius: 2,
                    p: 1,
                    mr: 2,
                  }}
                >
                  {React.cloneElement(stat.icon, { sx: { color: stat.color } })}
                </Box>
                <Typography variant="body2" color="text.secondary">
                  {stat.title}
                </Typography>
              </Box>
              <Typography variant="h5" fontWeight="bold">
                {stat.value}
              </Typography>
              <Typography
                variant="body2"
                sx={{
                  color: stat.change.startsWith('+') ? 'success.main' : 'error.main',
                  display: 'flex',
                  alignItems: 'center',
                }}
              >
                <TrendingUp sx={{ fontSize: 16, mr: 0.5 }} />
                {stat.change} from last month
              </Typography>
            </CardContent>
          </Card>
        </Grid>
      ))}
    </Grid>
  );
};

// =============================================================================
// RECENT ACTIVITY
// =============================================================================

const RecentActivity = () => {
  const activities = [
    { type: 'booking', message: 'New booking #12345 from John Doe', time: '2 min ago' },
    { type: 'vendor', message: 'Vendor "Lagos Catering" approved', time: '15 min ago' },
    { type: 'emergency', message: 'Emergency #567 resolved', time: '30 min ago' },
    { type: 'payment', message: 'Payment of ₦250,000 received', time: '1 hour ago' },
    { type: 'user', message: 'New user registration: jane@example.com', time: '2 hours ago' },
  ];

  return (
    <Card>
      <CardContent>
        <Typography variant="h6" gutterBottom>
          Recent Activity
        </Typography>
        <List dense>
          {activities.map((activity, index) => (
            <ListItem key={index}>
              <ListItemText
                primary={activity.message}
                secondary={activity.time}
              />
            </ListItem>
          ))}
        </List>
        <Button variant="text" fullWidth>
          View All Activity
        </Button>
      </CardContent>
    </Card>
  );
};

// =============================================================================
// PENDING APPROVALS
// =============================================================================

const PendingApprovals = () => {
  const approvals = [
    { id: 1, type: 'Vendor', name: 'Premium Events', status: 'pending' },
    { id: 2, type: 'Service', name: 'Deep Cleaning Package', status: 'pending' },
    { id: 3, type: 'Technician', name: 'Ibrahim Musa', status: 'pending' },
    { id: 4, type: 'Payout', name: '₦1,500,000 to Vendor #234', status: 'pending' },
  ];

  return (
    <Card>
      <CardContent>
        <Typography variant="h6" gutterBottom>
          Pending Approvals ({approvals.length})
        </Typography>
        <List dense>
          {approvals.map((item) => (
            <ListItem
              key={item.id}
              secondaryAction={
                <Box>
                  <Button size="small" color="success" sx={{ mr: 1 }}>
                    Approve
                  </Button>
                  <Button size="small" color="error">
                    Reject
                  </Button>
                </Box>
              }
            >
              <ListItemText
                primary={item.name}
                secondary={item.type}
              />
            </ListItem>
          ))}
        </List>
      </CardContent>
    </Card>
  );
};

// =============================================================================
// MAIN APP
// =============================================================================

function AdminDashboard() {
  const [mobileOpen, setMobileOpen] = useState(false);
  const [currentPage, setCurrentPage] = useState('Dashboard');

  const handleDrawerToggle = () => {
    setMobileOpen(!mobileOpen);
  };

  const drawer = (
    <Box>
      <Toolbar>
        <Typography variant="h6" noWrap component="div" fontWeight="bold">
          VendorPlatform
        </Typography>
      </Toolbar>
      <Divider />
      <List>
        {navigationItems.map((item, index) =>
          item.divider ? (
            <Divider key={index} sx={{ my: 1 }} />
          ) : (
            <ListItem
              button
              key={item.text}
              onClick={() => setCurrentPage(item.text)}
              sx={{
                borderRadius: 2,
                mx: 1,
                mb: 0.5,
                backgroundColor: currentPage === item.text ? 'primary.light' : 'transparent',
                color: currentPage === item.text ? 'white' : 'inherit',
                '&:hover': {
                  backgroundColor: currentPage === item.text ? 'primary.light' : 'action.hover',
                },
              }}
            >
              <ListItemIcon
                sx={{
                  color: currentPage === item.text ? 'white' : 'inherit',
                  minWidth: 40,
                }}
              >
                {item.icon}
              </ListItemIcon>
              <ListItemText primary={item.text} />
            </ListItem>
          )
        )}
      </List>
    </Box>
  );

  return (
    <ThemeProvider theme={theme}>
      <Box sx={{ display: 'flex' }}>
        <CssBaseline />
        
        {/* App Bar */}
        <AppBar
          position="fixed"
          sx={{
            width: { sm: `calc(100% - ${drawerWidth}px)` },
            ml: { sm: `${drawerWidth}px` },
            backgroundColor: 'background.paper',
            color: 'text.primary',
            boxShadow: '0 1px 3px rgba(0,0,0,0.1)',
          }}
        >
          <Toolbar>
            <IconButton
              color="inherit"
              edge="start"
              onClick={handleDrawerToggle}
              sx={{ mr: 2, display: { sm: 'none' } }}
            >
              <Menu />
            </IconButton>
            <Typography variant="h6" noWrap component="div" sx={{ flexGrow: 1 }}>
              {currentPage}
            </Typography>
            <IconButton color="inherit">
              <Badge badgeContent={4} color="error">
                <Notifications />
              </Badge>
            </IconButton>
            <Avatar sx={{ ml: 2, bgcolor: 'primary.main' }}>A</Avatar>
          </Toolbar>
        </AppBar>

        {/* Sidebar */}
        <Box
          component="nav"
          sx={{ width: { sm: drawerWidth }, flexShrink: { sm: 0 } }}
        >
          <Drawer
            variant="temporary"
            open={mobileOpen}
            onClose={handleDrawerToggle}
            ModalProps={{ keepMounted: true }}
            sx={{
              display: { xs: 'block', sm: 'none' },
              '& .MuiDrawer-paper': { boxSizing: 'border-box', width: drawerWidth },
            }}
          >
            {drawer}
          </Drawer>
          <Drawer
            variant="permanent"
            sx={{
              display: { xs: 'none', sm: 'block' },
              '& .MuiDrawer-paper': { boxSizing: 'border-box', width: drawerWidth },
            }}
            open
          >
            {drawer}
          </Drawer>
        </Box>

        {/* Main Content */}
        <Box
          component="main"
          sx={{
            flexGrow: 1,
            p: 3,
            width: { sm: `calc(100% - ${drawerWidth}px)` },
            mt: 8,
          }}
        >
          <DashboardStats />
          
          <Grid container spacing={3} sx={{ mt: 2 }}>
            <Grid item xs={12} md={6}>
              <RecentActivity />
            </Grid>
            <Grid item xs={12} md={6}>
              <PendingApprovals />
            </Grid>
          </Grid>
        </Box>
      </Box>
    </ThemeProvider>
  );
}

export default AdminDashboard;
