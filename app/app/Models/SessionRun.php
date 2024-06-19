<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\HasMany;

class SessionRun extends Model
{

    public function session(): BelongsTo
    {
        return $this->belongsTo(Session::class);
    }

    public function runs(): HasMany
    {
        return $this->hasMany(SessionItem::class, foreignKey: 'service_name', localKey: 'service_name')
            ->where('session_id', $this->session_id);
    }
}
