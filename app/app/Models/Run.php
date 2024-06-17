<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Factories\HasFactory;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

class Run extends Model
{
    protected $fillable = [
        'github_run_id',
        'status'
    ];

    public function session(): BelongsTo
    {
        return $this->belongsTo(Session::class);
    }
}
