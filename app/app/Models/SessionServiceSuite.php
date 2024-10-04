<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\HasMany;

class SessionServiceSuite extends Model
{
    protected $fillable = ['name', 'path'];

    protected $table = 'test_session_service_suites';

    public function service(): BelongsTo
    {
        return $this->belongsTo(SessionService::class, 'service_id');
    }

    public function tests(): HasMany
    {
        return $this->hasMany(SessionServiceSuiteTest::class, 'suite_id');
    }
}
