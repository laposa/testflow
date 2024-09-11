<?php

namespace App\Enums;

enum SessionActivityType: string
{
    case session_created = 'session_created';
    case run_dispatched = 'run_dispatched';
    case run_status_changed = 'run_status_changed';
    case review_requested = 'review_requested';
    case review_approved = 'review_approved';
    case review_rejected = 'review_rejected';
    case comment = 'comment';
}
