final class BackgroundPreferences {
  const BackgroundPreferences({
    required this.keepConnections,
    required this.notifications,
  });

  static const defaults = BackgroundPreferences(
    keepConnections: true,
    notifications: false,
  );

  final bool keepConnections;
  final bool notifications;

  BackgroundPreferences copyWith({
    bool? keepConnections,
    bool? notifications,
  }) => BackgroundPreferences(
    keepConnections: keepConnections ?? this.keepConnections,
    notifications: notifications ?? this.notifications,
  );

  Map<String, Object> toJson() => <String, Object>{
    'keepConnections': keepConnections,
    'notifications': notifications,
  };

  factory BackgroundPreferences.fromJson(Map<String, dynamic> json) =>
      BackgroundPreferences(
        keepConnections:
            json['keepConnections'] as bool? ?? defaults.keepConnections,
        notifications: json['notifications'] as bool? ?? defaults.notifications,
      );

  @override
  bool operator ==(Object other) =>
      other is BackgroundPreferences &&
      other.keepConnections == keepConnections &&
      other.notifications == notifications;

  @override
  int get hashCode => Object.hash(keepConnections, notifications);
}
