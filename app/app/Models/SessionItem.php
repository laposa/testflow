<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\BelongsToMany;
use Illuminate\Database\Eloquent\Relations\HasMany;

class SessionItem extends Model
{

    protected $fillable = [
        'repository_id',
        'workflow_id',
        'repository_name',
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

    public function runs(): BelongsToMany
    {
        return $this->belongsToMany(SessionRun::class, 'test_session_items_test_session_runs')
            ->withTimestamps();
    }
}
