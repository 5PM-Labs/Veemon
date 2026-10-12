import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:veemon_mobile/core/language/l10n_extension.dart';

import '../../domain/entities/user_entity.dart';
import '../cubits/session_cubit.dart';
import '../cubits/session_state.dart';

class ProfileScreen extends StatelessWidget {
  const ProfileScreen({super.key});

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    final user = context.select<SessionCubit, UserEntity?>(
      (cubit) => switch (cubit.state) {
        SessionAuthenticated(:final user) => user,
        _ => null,
      },
    );
    return Scaffold(
      appBar: AppBar(
        title: Text(l10n.profileTitle),
        actions: [
          IconButton(
            tooltip: l10n.profileLogout,
            icon: const Icon(Icons.logout),
            onPressed: () => context.read<SessionCubit>().logout(),
          ),
        ],
      ),
      body: user == null ? const SizedBox.shrink() : _ProfileDetails(user),
    );
  }
}

class _ProfileDetails extends StatelessWidget {
  const _ProfileDetails(this.user);

  final UserEntity user;

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    return ListView(
      children: [
        ListTile(
          title: Text(
            user.name,
            style: Theme.of(context).textTheme.headlineSmall,
          ),
        ),
        ListTile(
          title: Text(l10n.profileEmailLabel),
          subtitle: Text(user.email),
        ),
        if (user.phone.isNotEmpty)
          ListTile(
            title: Text(l10n.profilePhoneLabel),
            subtitle: Text(user.phone),
          ),
        ListTile(
          title: Text(l10n.profileStatusLabel),
          subtitle: Text(user.status),
        ),
      ],
    );
  }
}
