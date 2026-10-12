import 'package:flutter/material.dart';
import 'package:veemon_mobile/core/widgets/loading_indicator.dart';

class SplashScreen extends StatelessWidget {
  const SplashScreen({super.key});

  @override
  Widget build(BuildContext context) =>
      const Scaffold(body: LoadingIndicator());
}
