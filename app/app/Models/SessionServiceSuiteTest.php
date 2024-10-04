<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

class SessionServiceSuiteTest extends Model
{
    protected $fillable = ['name', 'path'];

    protected $table = 'test_session_service_suite_tests';

    public function suite(): BelongsTo
    {
        return $this->belongsTo(SessionServiceSuite::class, 'suite_id');
    }
}
