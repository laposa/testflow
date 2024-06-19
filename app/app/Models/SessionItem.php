<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\HasMany;

class SessionItem extends Model
{

    protected $fillable = [
        'service_name',
        'suite_name',
        'test_name',
        'service_url',
    ];

    protected $table = 'test_session_items';

    public function session(): BelongsTo
    {
        return $this->belongsTo(Session::class);
    }

    public function runs(): HasMany
    {
        return $this->hasMany(SessionRun::class, foreignKey: 'service_name', localKey: 'service_name')
            ->where('session_id', $this->session_id);
    }
}
