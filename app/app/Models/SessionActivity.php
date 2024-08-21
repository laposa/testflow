<?php

namespace App\Models;

use App\Enums\SessionActivityType;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

class SessionActivity extends Model
{
    protected $fillable = ['user_id', 'type', 'body', 'metadata'];

    protected $table = 'session_activity_log';

    protected function casts(): array
    {
        return [
            'type' => SessionActivityType::class,
            'metadata' => 'array',
        ];
    }

    public function session(): BelongsTo
    {
        return $this->belongsTo(Session::class);
    }

    public function user(): BelongsTo
    {
        return $this->belongsTo(User::class);
    }
}
